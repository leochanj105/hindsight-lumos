package agent

import (
    "fmt"
    "sync"
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

    /* Used by the TriggerManager to tell the cache of triggered traces */
    cache_triggers <-chan uint64
    cache_expired_triggers <-chan uint64

    /* Used by TriggerManager after reporting buffers, to make them available again */
    available_buffers <-chan int

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

    // Channels for receiving triggers and redirecting triggered traces
    triggered           map[uint64]bool    // Trace IDs that have been triggered
    triggers            chan uint64        // Receive new triggered trace IDs
    expired_triggers    chan uint64        // Trace IDs that are now expired
    triggered_data      <-chan *TraceData  // Used to send data for triggered traces
}

func InitAgent(fname string, delay int, capacity int) *Agent {
    var agent Agent
    // agent.api = memory.InitGoAgentAPI(fname)

    var cache TraceCache
    cache.capacity = capacity
    cache.eviction_batch_size = 20 // Hard code to some value for now
    cache.buf_count = 0
    cache.lru = list.New()
    cache.data = make(map[uint64]*list.Element)
    cache.triggered = make(map[uint64]bool)
    cache.triggers = make(chan uint64)
    cache.expired_triggers = make(chan uint64)
    agent.cache = &cache;

    var triggers TriggerManager
    triggers.local_triggers = make(chan uint64)
    triggers.remote_triggers = make(chan uint64)
    triggers.new_trace_data = make(chan *TraceData)
    // TODO: get other components channels
    triggers.timers = make(map[uint64](chan bool))
    triggers.trace_ids = treeset.NewWithIntComparator()
    triggers.unreported_data = make(map[uint64]*TraceData)
    agent.trigger_manager = &triggers

    cache.triggered_data = triggers.new_trace_data
    triggers.cache_triggers = cache.triggers
    triggers.cache_expired_triggers = cache.expired_triggers

    return &agent
}

func (agent *Agent) Run() {
    wg := new(sync.WaitGroup)
    wg.Add(1)
    go func() {
        agent.trigger_manager.Run()
        wg.Done()
    }()
    wg.Wait()
}

func (tm *TriggerManager) Run() {
    for {
        select {
        case trace_id := <-tm.local_triggers:
            fmt.Println("Local trigger", trace_id)
        case trace_id := <-tm.remote_triggers:
            fmt.Println("Remote trigger", trace_id)
        }
    }
}