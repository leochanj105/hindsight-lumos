package agent

import (
    "fmt"
    "sync"
    "context"
    "time"
    "log"
    "net"
    "container/list"
    "github.com/emirpasic/gods/sets/treeset"
    "github.com/geraldleizhang/hindsight/agent/pkg/util"
    "github.com/geraldleizhang/hindsight/agent/pkg/memory"
    "github.com/geraldleizhang/hindsight/agent/pkg/datapb"
    "google.golang.org/grpc"
)

type TraceData struct {
    Request_id  uint64
    Buffers     []int
    Breadcrumbs []string
}

func (trace *TraceData) Add(other *TraceData) {
    if other == nil {
        return
    }
    trace.Buffers = append(trace.Buffers, other.Buffers...)
    trace.Breadcrumbs = append(trace.Breadcrumbs, other.Breadcrumbs...)
}

type Agent struct {
    api *memory.GoAgentAPI  // API to the shared memory
    trigger_delay uint64     // Used for experiments; hard-coded delay before trigger fires
    cache *TraceCache
    trigger_manager *TriggerManager
}

/*
Trigger manager manages all state to do with triggers
that are fired both locally and received from the log
collector
*/
type TriggerManager struct {
    api *memory.GoAgentAPI  // API to the shared memory

    // Incoming from grpc handler
    remote_triggers chan uint64

    // Channel for receiving triggered trace data from cache
    new_trace_data chan *TraceData

    /* Every triggered trace will initiate a timer that eventually evicts the trace
    from the trigger manager.  These channels are used to reset the timeout when 
    new trace data arrives */
    triggered      map[uint64](chan struct{})
    timeouts    chan uint64

    /* Stores the trace IDs of all triggered traces, until they expire.
    These are sorted, in case triggering becomes a bottleneck; low trace IDs
    get reported first*/
    unreported_trace_ids *treeset.Set
    has_unreported_trace_ids chan struct{}

    /* TraceData that hasn't been reported yet.  A trace ID will remain in
    this map until it expires, but any reported buffers get immediately 
    cleared from the list */
    unreported_data map[uint64]*TraceData

    // Incoming from shm (GoAgentAPI)
    available   chan<- []int                    // to send evicted buffers to shm available queue
    triggers    <-chan []memory.Trigger         // triggers from shm

    /* Used by the TriggerManager to tell the cache of triggered traces and freed bufs */
    cache_triggers chan<- uint64                // From TraceCache
    cache_expired_triggers chan<- uint64        // From TraceCache
    cache_notify_available_buffers chan<- int   // From TraceCache
}

type CacheStats struct {
    complete_batches int
    complete_buffers int
}

/*
TraceCache simply caches and evicts buffers.  If the TriggerManager triggers
a trace ID, the TraceCache will send all data to the TriggerManager and
redirect all future buffers to the TriggerManager until told otherwise
*/
type TraceCache struct {
    // Constants for deciding when to evict 
    capacity            int // Above this threshold, we should evict
    buffer_size         int // Size of buffers in the cache
    eviction_batch_size int // Each eviction should aim for this many buffers
    buf_count           int // Current number of cached buffers. Not the same

    // TraceData stored and managed by the TraceCache
    lru         *list.List
    data        map[uint64]*list.Element // e.Value.(*TraceData)

    // Incoming from shm (GoAgentAPI)
    available   chan<- []int                    // to send evicted buffers to shm available queue
    complete    <-chan memory.CompleteBatch     // buffers from shm complete queue
    breadcrumbs <-chan memory.BreadcrumbBatch   // breadcrumbs from shm

    // Used internally to trigger eviction
    eviction_required chan struct{}

    // Stats for logging
    stats       CacheStats
    last_print  uint64
    print_every uint64
    next_print *time.Timer

    // Triggered and expired traces
    triggered           map[uint64]struct{}// Trace IDs that have been triggered
    triggers            chan uint64        // Receive new triggered trace IDs
    expired_triggers    chan uint64        // Trace IDs that are now expired
    triggered_data      chan<- *TraceData  //   From TriggerManager
    notify_available_buffers chan int      // Notify of change in cache capacity
}

func InitAgent(fname string, trigger_delay uint64) *Agent {
    var api *memory.GoAgentAPI
    api = memory.InitGoAgentAPI(fname)    

    var cache TraceCache
    cache.capacity = (4 * api.Capacity()) / 5 // TODO: not hardcoded
    cache.buffer_size = api.BufferSize()
    cache.eviction_batch_size = 20 // Hard code to some value for now
    cache.buf_count = 0
    cache.lru = list.New()
    cache.data = make(map[uint64]*list.Element)
    cache.triggered = make(map[uint64]struct{})
    cache.triggers = make(chan uint64, 10000)
    cache.expired_triggers = make(chan uint64, 10000)
    cache.notify_available_buffers = make(chan int, 10000)
    cache.eviction_required = make(chan struct{}, 1000)
    cache.print_every = 5000
    cache.next_print = time.NewTimer(1 * time.Millisecond)

    var triggers TriggerManager
    triggers.api = api
    triggers.remote_triggers = make(chan uint64, 1000)
    triggers.new_trace_data = make(chan *TraceData, 10000)
    triggers.triggered = make(map[uint64](chan struct{}))
    triggers.timeouts = make(chan uint64, 10000)
    triggers.unreported_trace_ids = treeset.NewWithIntComparator()
    triggers.has_unreported_trace_ids = make(chan struct{}, 1000)
    triggers.unreported_data = make(map[uint64]*TraceData)

    //// Link up channels

    cache.triggered_data = triggers.new_trace_data
    cache.available = api.Available
    cache.complete = api.Complete
    cache.breadcrumbs = api.Breadcrumbs

    triggers.available = api.Available
    triggers.cache_triggers = cache.triggers
    triggers.cache_expired_triggers = cache.expired_triggers
    triggers.cache_notify_available_buffers = cache.notify_available_buffers

    // Hack-ish here to delay triggers
    if trigger_delay == 0 {
        triggers.triggers = api.Triggers
    } else {
        proxy := make(chan []memory.Trigger, 10000)
        triggers.triggers = proxy
        go func() {
            for {
                select {
                case fired := <-api.Triggers: {
                    go func() {
                        time.Sleep(time.Duration(trigger_delay) * time.Nanosecond)
                        proxy <- fired
                    }()
                }
                }
            }
        }()
    }

    //// Create final agent

    var agent Agent
    agent.api = api
    agent.trigger_delay = trigger_delay
    agent.cache = &cache;
    agent.trigger_manager = &triggers

    fmt.Println("Go Agent cache capacity", cache.capacity)

    return &agent
}

func (agent *Agent) Run(ctx context.Context) {
    wg := new(sync.WaitGroup)
    wg.Add(4)
    go func() {
        agent.cache.Run(ctx)
        wg.Done()
    }()
    go func() {
        agent.trigger_manager.Run(ctx)
        wg.Done()
    }()
    go func() {
        agent.trigger_manager.RunGRPCServer()
        wg.Done()
    }()
    go func() {
        agent.api.Run(ctx)
        wg.Done()
    }()
    // LEI TODO: grpc server for remote triggers
    wg.Wait()
}

func (tm *TriggerManager) addTraceData(trace *TraceData) {
    trace_id := trace.Request_id

    if canceller, ok := tm.triggered[trace_id]; ok {
        /* This trace is still triggered. Cancel the old timer */
        canceller <- struct{}{}
    } else {
        /* This trace is not triggered. Ditch the buffers, remind
        the cache it's not triggered, and leave */
        tm.cache_notify_available_buffers <- len(trace.Buffers)
        tm.available <- trace.Buffers
        tm.cache_expired_triggers <- trace_id
        return
    }
    
    if existing, ok := tm.unreported_data[trace_id]; ok {
        /* There's some unreported data for this trace */
        existing.Add(trace)
    } else {
        /* Add the new trace data */
        tm.unreported_data[trace_id] = trace
        tm.unreported_trace_ids.Add(int(trace_id))
        tm.has_unreported_trace_ids <- struct{}{}
    }

    // Set a new timeout for the trace
    canceller := make(chan struct{})
    tm.triggered[trace_id] = canceller
    go func() {
        // TODO this timeout can be much higher, e.g. minutes
        select {
        case <- time.After(3 * time.Second):
            tm.timeouts <- trace_id
        case <- canceller:
            return
        }
    }()
}

/* Sets a trace as triggered.  This is called by the client
invoking trigger over shm, and by the collector sending a 
trigger to us.  We will send all current and future data
for this trace ID to the collector.  We will stop sending 
data for this trace after 60 seconds.  */
func (tm *TriggerManager) addTrigger(trace_id uint64) {
    log.Println("Received trigger for", trace_id)
    if canceller, ok := tm.triggered[trace_id]; ok {
        /* Already triggered; cancel old timeout */
        canceller <- struct{}{}
    }

    // Set a new timeout for the trace
    canceller := make(chan struct{})
    tm.triggered[trace_id] = canceller
    go func() {
        // TODO this timeout can be much higher, e.g. minutes
        select {
        case <- time.After(3 * time.Second):
            tm.timeouts <- trace_id
        case <- canceller:
            return
        }
    }()

    // Notify cache
    tm.cache_triggers <- trace_id    
}

/* Sets a trace as untriggered and stops collecting its data */
func (tm *TriggerManager) unTrigger(trace_id uint64) {
    if _, ok := tm.triggered[trace_id]; ok {
        /* Delete the trigger */
        delete(tm.triggered, trace_id)
    }

    if trace, ok := tm.unreported_data[trace_id]; ok {
        /* Return the buffers */
        tm.cache_notify_available_buffers <- len(trace.Buffers)
        tm.available <- trace.Buffers

        /* Delete */
        delete(tm.unreported_data, trace_id)
        tm.unreported_trace_ids.Remove(int(trace_id))
    }

    /* Notify the cache to untrigger */
    tm.cache_expired_triggers <- trace_id
}

/* Reports trace data to the collector */
func (tm *TriggerManager) reportNext(lc datapb.CollectorClient) {
    it := tm.unreported_trace_ids.Iterator()
    if !it.First() {
        return
    }

    // Get the next trace to report
    trace_id := uint64(it.Value().(int))
    trace := tm.unreported_data[trace_id]

    // Remove from unreported data
    tm.unreported_trace_ids.Remove(int(trace_id))
    delete(tm.unreported_data, trace_id)


    fmt.Printf("Reporting trace %d with %d buffers, breadcrumbs: ", trace_id, len(trace.Buffers))
    for i, addr := range trace.Breadcrumbs {
        fmt.Printf("(%d: %s) ", i, addr)
    }
    fmt.Printf("\n")

    var entry []int32
    var trace_data []byte
    var addrs []string
    addrs = append(addrs, util.Server_addr+":"+util.Server_port)

    for _, buffer_id := range trace.Buffers {
        entry = append(entry, int32(buffer_id))
        data := tm.api.GetBuffer(buffer_id)
        trace_data = append(trace_data, data...)
        addrs = append(addrs, trace.Breadcrumbs...)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Nanosecond)
    defer cancel()

    _, err := lc.Report(ctx, &datapb.Trace{
        RequestId: int64(trace_id),
        Entry:     entry,
        Trace:     trace_data,
        Addrs:     addrs})

    if err != nil {
        fmt.Println("report", err)
        // return
    }
}

/* gRPC requests from Log collector */
func (tm *TriggerManager) Request(ctx context.Context, in *datapb.RequestID) (*datapb.CallRet, error) {
    request_ids := in.Rid
    for _, request_id := range request_ids {
        tm.remote_triggers <- uint64(request_id)
    }
    return &datapb.CallRet{Callret: true}, nil
}


func (tm *TriggerManager) Run(ctx context.Context) {
    fmt.Println("TriggerManager goroutine running")
    conn, err := grpc.Dial(util.LC_addr+":"+util.LC_port, 
                            grpc.WithInsecure(), 
                            grpc.WithTimeout(100000000*time.Nanosecond))
    if err != nil {
        fmt.Println("dial", util.LC_addr+":"+util.LC_port, err)
        return
    }
    fmt.Println("TriggerManager connected to", util.LC_addr+":"+util.LC_port)
    defer conn.Close()

    collector := datapb.NewCollectorClient(conn)
    for {
        select {
        case triggers := <-tm.triggers:
            for _, trigger := range triggers {
                tm.addTrigger(trigger.Request_id)
            }
        case trace_id := <-tm.remote_triggers:
            tm.addTrigger(trace_id)
        case trace_data := <-tm.new_trace_data:
            tm.addTraceData(trace_data)
        case trace_id := <-tm.timeouts:
            tm.unTrigger(trace_id)
        case <- tm.has_unreported_trace_ids:
            tm.reportNext(collector)
        case <- ctx.Done():
            return
        }
    }
}

func (tm *TriggerManager) RunGRPCServer() {
    for true {
        lis, err := net.Listen("tcp", ":"+util.Server_port)
        if err != nil {
            log.Fatalf("failed to listen: %v", err)
        }
        s := grpc.NewServer()
        datapb.RegisterAgentServer(s, tm)
        if err := s.Serve(lis); err != nil {
            log.Fatalf("failed to serve: %v", err)
        }
    }
}

func (cache* TraceCache) evictionRequired() bool {
    return cache.buf_count > cache.capacity
}

/* Check if the cache is over capacity, and evict some buffers if so */
func (cache* TraceCache) checkEviction() bool {
    if !cache.evictionRequired() {
        return false
    }

    // Figure out what should be evicted
    var evicted_buffers []int
    for cache.buf_count > cache.capacity {
        // Get our victim
        entry := cache.lru.Back()
        trace := entry.Value.(*TraceData)

        // Remove
        delete(cache.data, trace.Request_id)
        cache.lru.Remove(entry)

        // Manage buffers
        cache.buf_count -= len(trace.Buffers)
        evicted_buffers = append(evicted_buffers, trace.Buffers...)
    }

    // Make the evicted buffers available
    cache.available <- evicted_buffers

    return true
}

/* Add some buffers to the cache */
func (cache *TraceCache) addCompletedBuffers(batch memory.CompleteBatch) {
    for trace_id, buffer_ids := range batch {
        /* Ignore trace ID 0 */
        if trace_id == 0 {
            cache.available <- buffer_ids
            continue
        }

        cache.buf_count += len(buffer_ids)
        cache.stats.complete_buffers += len(buffer_ids)

        if _, ok := cache.triggered[trace_id]; ok {
            /* Buffer for a triggered request */
            td := &TraceData{trace_id, buffer_ids, nil}
            cache.triggered_data <- td
            continue
        }

        if entry, ok := cache.data[trace_id]; ok {
            /* Buffer for a trace already in cache */
            td := entry.Value.(*TraceData)
            td.Buffers = append(td.Buffers, buffer_ids...)
            cache.lru.MoveToFront(entry)
            continue
        }

        /* Buffer for a trace not in cache */
        td := &TraceData{trace_id, buffer_ids, nil}
        entry := cache.lru.PushFront(td)
        cache.data[trace_id] = entry
    }

    /* Trigger eviction if above threshold */
    if cache.evictionRequired() {
        cache.eviction_required <- struct{}{}
    }


    cache.stats.complete_batches++
}

func (cache *TraceCache) print() {
    now := uint64(time.Now().UnixNano())
    count := cache.stats.complete_batches
    sum := cache.stats.complete_buffers
    tput := float32(uint64(sum) * 1000000000) / float32(now - cache.last_print)
    tput_mb := (tput * float32(cache.buffer_size)) / (1024 * 1024)
    var batchsize float32
    if count == 0 {
        batchsize = 0
    } else {
        batchsize = float32(sum) / float32(count)
    }
    fmt.Printf("%.0f MB/s (%.0f bufs/s, %d bufs total), Avg batch %.1f\n", tput_mb, tput, sum, batchsize)
    cache.last_print = now
    cache.stats.complete_batches = 0
    cache.stats.complete_buffers = 0
}

/* Add some breadcrumbs to the cache */
func (cache *TraceCache) addBreadcrumbs(batch memory.BreadcrumbBatch) {
    for trace_id, breadcrumbs := range batch {
        if _, ok := cache.triggered[trace_id]; ok {
            /* Buffer for a triggered request */
            td := &TraceData{trace_id, nil, breadcrumbs}
            cache.triggered_data <- td
            continue
        }

        if entry, ok := cache.data[trace_id]; ok {
            /* Buffer for a trace already in cache */
            td := entry.Value.(*TraceData)
            td.Breadcrumbs = append(td.Breadcrumbs, breadcrumbs...)
            cache.lru.MoveToFront(entry)
            continue
        }

        /* Buffer for a trace not in cache */
        td := &TraceData{trace_id, nil, breadcrumbs}
        entry := cache.lru.PushFront(td)
        cache.data[trace_id] = entry
    }
}

func (cache *TraceCache) Run(ctx context.Context) {
    fmt.Println("TraceCache goroutine running")
    for {
        select {
        case <- ctx.Done():
            return
        case delta := <-cache.notify_available_buffers: {
            /* The trigger manager notifies us that it returned `delta` 
            buffers to the available queue, and we can increase cache capacity
            as a result */
            cache.buf_count -= delta
        }
        case trace_id := <-cache.triggers: {
            /* The trigger manager notifies us that `trace_id` is triggered.
            The cache stops caching buffers for this trace_id, sends any
            existing buffers to the trigger manager, and forwards all future
            buffers directly to the trigger manager */

            // Ignore any trace_ids that are already triggered
            if _, ok := cache.triggered[trace_id]; ok {
                continue;
            }

            // Mark as cached
            cache.triggered[trace_id] = struct{}{};

            // If any data exists, send to the trigger manager
            if entry, ok := cache.data[trace_id]; ok {
                trace_data := entry.Value.(*TraceData)

                delete(cache.data, trace_id)
                cache.lru.Remove(entry)

                cache.triggered_data <- trace_data
            }

        }
        case trace_id := <-cache.expired_triggers: {
            /* The trigger manager notifies us that `trace_id` is no longer
            triggered.  The cache reverts to the standard handling, if this
            `trace_id` is seen again */

            delete(cache.triggered, trace_id)
        }
        case buffers := <-cache.complete: {
            /* Received some buffers from the shm complete queue */
            cache.addCompletedBuffers(buffers)
        }
        case breadcrumbs := <-cache.breadcrumbs: {
            /* Received some breadcrumbs from the shm breadcrumbs queue */
            cache.addBreadcrumbs(breadcrumbs)
        }
        case <- cache.eviction_required: {
            cache.checkEviction()
        }
        case <- cache.next_print.C: {
            cache.print()
            cache.next_print.Reset(1000 * time.Millisecond)
        }
        }
    }
}
