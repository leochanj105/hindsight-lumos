package agent

import (
    "fmt"
    "sync"
    "context"
    "time"
    "container/list"
    "github.com/emirpasic/gods/sets/treeset"
    "github.com/geraldleizhang/hindsight/agent/pkg/memory"
)

type TraceData struct {
    Request_id  uint64
    Buffers     []int
    Breadcrumbs []string
}

type Agent struct {
    api *memory.GoAgentAPI  // API to the shared memory
    trigger_delay int64     // Used for experiments; hard-coded delay before trigger fires
    cache *TraceCache
    trigger_manager *TriggerManager
}

/*
Trigger manager manages all state to do with triggers
that are fired both locally and received from the log
collector
*/
type TriggerManager struct {
    local_triggers chan uint64      // Channel used to receive local triggers
    remote_triggers chan uint64     // Channel used to receive remote triggers
    new_trace_data chan *TraceData  // Channel used to receive data from the cache

    /* Used by the TriggerManager to tell the cache of triggered traces and freed bufs */
    cache_triggers <-chan uint64                // From TraceCache
    cache_expired_triggers <-chan uint64        // From TraceCache
    cache_notify_available_buffers <-chan int   // From TraceCache


    /* Used by TriggerManager after reporting buffers, to make them available again */
    available_buffers <-chan []int

    /* Every triggered trace will initiate a timer that eventually evicts the trace
    from the trigger manager.  These channels are used to reset the timeout when 
    new trace data arrives */
    timers map[uint64](chan bool)    

    /* Stores the trace IDs of all triggered traces, until they expire.
    These are sorted, in case triggering becomes a bottleneck; low trace IDs
    get reported first*/
    trace_ids *treeset.Set

    /* TraceData that hasn't been reported yet.  A trace ID will remain in
    this map until it expires, but any reported buffers get immediately 
    cleared from the list */
    unreported_data map[uint64]*TraceData
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
    eviction_batch_size int // Each eviction should aim for this many buffers
    buf_count           int // Current number of cached buffers. Not the same

    // TraceData stored and managed by the TraceCache
    lru         *list.List
    data        map[uint64]*list.Element // e.Value.(*TraceData)

    // Incoming from shm (GoAgentAPI)
    available   chan<- []int                    // to send evicted buffers to shm available queue
    complete    <-chan memory.CompleteBatch     // buffers from shm complete queue
    breadcrumbs <-chan memory.BreadcrumbBatch   // breadcrumbs from shm

    // Stats for logging
    stats CacheStats
    last_print uint64

    // Triggered and expired traces
    triggered           map[uint64]bool    // Trace IDs that have been triggered
    triggers            chan uint64        // Receive new triggered trace IDs
    expired_triggers    chan uint64        // Trace IDs that are now expired
    triggered_data      chan<- *TraceData  //   From TriggerManager
    notify_available_buffers chan int      // Notify of change in cache capacity
}

func InitAgent(fname string, delay int) *Agent {
    var api *memory.GoAgentAPI
    api = memory.InitGoAgentAPI(fname)    

    var cache TraceCache
    cache.capacity = (4 * api.Capacity()) / 5 // TODO: not hardcoded
    cache.eviction_batch_size = 20 // Hard code to some value for now
    cache.buf_count = 0
    cache.lru = list.New()
    cache.data = make(map[uint64]*list.Element)
    cache.triggered = make(map[uint64]bool)
    cache.triggers = make(chan uint64)
    cache.expired_triggers = make(chan uint64)
    cache.notify_available_buffers = make(chan int)

    var triggers TriggerManager
    triggers.local_triggers = make(chan uint64)
    triggers.remote_triggers = make(chan uint64)
    triggers.new_trace_data = make(chan *TraceData)
    triggers.timers = make(map[uint64](chan bool))
    triggers.trace_ids = treeset.NewWithIntComparator()
    triggers.unreported_data = make(map[uint64]*TraceData)

    //// Link up channels

    cache.triggered_data = triggers.new_trace_data
    cache.available = api.Available
    cache.complete = api.Complete
    cache.breadcrumbs = api.Breadcrumbs

    triggers.cache_triggers = cache.triggers
    triggers.cache_expired_triggers = cache.expired_triggers
    triggers.cache_notify_available_buffers = cache.notify_available_buffers

    //// Create final agent

    var agent Agent
    agent.api = api
    agent.cache = &cache;
    agent.trigger_manager = &triggers

    fmt.Println("Go Agent cache capacity", cache.capacity)

    return &agent
}

func (agent *Agent) Run(ctx context.Context) {
    wg := new(sync.WaitGroup)
    wg.Add(3)
    go func() {
        agent.cache.Run(ctx)
        wg.Done()
    }()
    go func() {
        agent.trigger_manager.Run(ctx)
        wg.Done()
    }()
    go func() {
        agent.api.Run(ctx)
        wg.Done()
    }()
    wg.Wait()
}

func (tm *TriggerManager) Run(ctx context.Context) {
    fmt.Println("TriggerManager goroutine running")
    for {
        select {
        case trace_id := <-tm.local_triggers:
            fmt.Println("Local trigger", trace_id)
        case trace_id := <-tm.remote_triggers:
            fmt.Println("Remote trigger", trace_id)
        case <- ctx.Done():
            return
        }
    }
}

/* Check if the cache is over capacity, and evict some buffers if so */
func (cache* TraceCache) checkEviction() bool {
    if cache.buf_count <= cache.capacity {
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

    cache.stats.complete_batches++

    var print_every uint64
    print_every = 1000000000
    now := uint64(time.Now().UnixNano())
    if ((now - cache.last_print) > print_every) {
        count := cache.stats.complete_batches
        sum := cache.stats.complete_buffers
        tput := (uint64(sum) * print_every) / (now - cache.last_print)
        batchsize := float32(sum) / float32(count)
        fmt.Println("Throughput:", tput, "Average batch:", batchsize)
        cache.last_print = now
        cache.stats.complete_batches = 0
        cache.stats.complete_buffers = 0
    }
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
            cache.triggered[trace_id] = true;

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
        default: {
            /* Default case: check if eviction is needed */
            cache.checkEviction()
        }
        }
    }
}