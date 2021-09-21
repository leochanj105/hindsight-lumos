package agent

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/geraldleizhang/hindsight/agent/pkg/memory"
	"github.com/geraldleizhang/hindsight/agent/pkg/util"
	"github.com/juju/ratelimit"
)

/* All-in-one agent implementation */

type TraceData struct {
	trace_id    uint64
	buffers     []int
	breadcrumbs []string

	needs_reporting bool
	triggered_by    []*TriggeredTrace

	last_modified time.Time

	e *list.Element // Node in regular LRU
}

type TriggeredTrace struct {
	trigger *TriggerState
	e       *list.Element // Node in trigger's LRU
}

type TriggerState struct {
	trigger_id int
	lru        *list.List // LRU of traces triggered by this trigger
	unreported *util.TreeNode
	size       int // Number of unreported buffers
	vt         int // Virtual time used for fair sharing

	limiter *ratelimit.Bucket // Rate limiter

	metrics TriggerMetrics
}

type Agent struct {
	api       *memory.GoAgentAPI // API to the shared memory
	reporting *Reporting         // Interface to LogCollector

	// Constants for deciding when to evict
	cache_capacity int // Above this threshold, we should evict
	cache_size     int // Current number of cached buffers.

	triggered_capacity int // Above this threshold we evict
	triggered_size     int // Current number of triggered buffers.

	lru *list.List // LRU of untriggered traces

	triggered              map[int]*TriggerState
	triggered_lru_size     int // Number of triggered traces
	triggered_lru_capacity int // Maximum number of triggered traces

	buffer_size         int // Size of buffers in the cache
	eviction_batch_size int // Each eviction should aim for this many buffers

	// TraceData stored and managed by the TraceCache
	data map[uint64]*TraceData // e.Value.(*TraceData)

	/* Wraps api.Triggers, possibly adding a delay */
	triggers <-chan []memory.Trigger // triggers from shm

	metrics AgentMetrics

	vc int // Virtual clock used for fair sharing

	now time.Time
}

func (agent *Agent) getOrCreateTriggerState(trigger_id int) *TriggerState {
	if trigger, ok := agent.triggered[trigger_id]; ok {
		return trigger
	}

	var trigger TriggerState
	trigger.trigger_id = trigger_id
	trigger.lru = list.New()
	trigger.unreported = util.InitPartialPriorityTree()
	trigger.vt = agent.vc
	agent.triggered[trigger_id] = &trigger

	return &trigger
}

func InitAgent(fname string, trigger_delay uint64, rate_limit float64, per_trigger_rate_limits map[int]float64) *Agent {
	fmt.Println("Init agent", fname)
	fmt.Printf("  Trigger delay %d nanoseconds\n", trigger_delay)
	fmt.Printf("  Reporting rate limit %.2f MB/s\n", rate_limit)
	for trigger_id, rate := range per_trigger_rate_limits {
		fmt.Printf("    -Trigger %d rate limit %.2f MB/s\n", trigger_id, rate)
	}

	var agent Agent
	agent.api = memory.InitGoAgentAPI(fname)
	agent.reporting = InitReporting(agent.api, rate_limit)

	agent.cache_capacity = (4 * agent.api.Capacity()) / 5 // TODO: not hardcoded
	agent.cache_size = 0

	agent.triggered_capacity = agent.cache_capacity / 2 // TODO: not hardcoded
	agent.triggered_size = 0

	agent.lru = list.New()
	agent.triggered = make(map[int]*TriggerState)
	agent.triggered_lru_capacity = agent.cache_capacity

	agent.buffer_size = agent.api.BufferSize()
	agent.eviction_batch_size = agent.cache_capacity / 100 // Hard code to some value for now

	agent.data = make(map[uint64]*TraceData)

	if trigger_delay == 0 {
		agent.triggers = agent.api.Triggers
	} else {
		proxy := make(chan []memory.Trigger, 10000)
		agent.triggers = proxy
		go func() {
			for {
				select {
				case fired := <-agent.api.Triggers:
					go func() {
						time.Sleep(time.Duration(trigger_delay) * time.Nanosecond)
						proxy <- fired
					}()
				}
			}
		}()
	}

	/* Eagerly create trigger state for specified rate limits */
	for trigger_id, limit := range per_trigger_rate_limits {
		trigger := agent.getOrCreateTriggerState(trigger_id)
		limit_bytes := limit * 1024 * 1024
		trigger.limiter = ratelimit.NewBucketWithRate(limit_bytes, int64(limit_bytes))
	}

	fmt.Println("Go Agent cache capacity", agent.cache_capacity)

	return &agent
}

/* Should we drop unused trace IDs from trigger history? */
func (agent *Agent) maybeEvictTriggeredLRU() {
	if agent.triggered_lru_size <= agent.triggered_lru_capacity {
		return
	}

	// Find the trigger consuming the most LRU capacity
	var trigger *TriggerState
	for _, candidate := range agent.triggered {
		if trigger == nil || candidate.lru.Len() > trigger.lru.Len() {
			trigger = candidate
		}
	}

	// Figure out how much should be evicted
	target_lru_size := agent.triggered_lru_capacity - agent.eviction_batch_size

	// Only evict from largest trigger each time; don't try to be clever
	for agent.triggered_lru_size > target_lru_size && trigger.lru.Len() > 0 {
		// Get our victim
		entry := trigger.lru.Back()
		trace := entry.Value.(*TraceData)

		// Untrigger and possibly delete; this should never return buffers
		agent.untriggered(trace, trigger)
	}
}

/* Check if the cache is over capacity, and evict some buffers if so */
func (agent *Agent) maybeEvictTriggered() {
	agent.maybeEvictTriggeredLRU()

	if agent.triggered_size <= agent.triggered_capacity {
		return
	}

	// Find the trigger consuming the most capacity
	var trigger *TriggerState
	for _, candidate := range agent.triggered {
		if trigger == nil || candidate.size > trigger.size {
			trigger = candidate
		}
	}

	// Figure out how much should be evicted
	to_evict := agent.triggered_size - agent.triggered_capacity
	if to_evict < agent.eviction_batch_size {
		to_evict = agent.eviction_batch_size
	}

	// Only evict from largest trigger each time; don't try to be clever
	var evicted_buffers []int
	for len(evicted_buffers) < to_evict && trigger.unreported.Size() > 0 {
		// Get our victim
		trace_id := trigger.unreported.PopNearMax()

		if trace, ok := agent.data[trace_id]; ok {
			bufs := agent.untriggered(trace, trigger)
			evicted_buffers = append(evicted_buffers, bufs...)
		}
	}

	// Make the evicted buffers available
	if len(evicted_buffers) > 0 {
		agent.api.Available <- evicted_buffers
	}
}

func (agent *Agent) maybeEvict() {
	// Evict triggered first
	agent.maybeEvictTriggered()

	if agent.cache_size < agent.cache_capacity {
		return
	}

	// Figure out how much should be evicted
	to_evict := agent.cache_size - agent.cache_capacity
	if to_evict < agent.eviction_batch_size {
		to_evict = agent.eviction_batch_size
	}

	var evicted_buffers []int
	for len(evicted_buffers) < to_evict && agent.lru.Len() > 0 {
		// Get our victim
		entry := agent.lru.Back()
		trace := entry.Value.(*TraceData)

		// Remove
		delete(agent.data, trace.trace_id)
		agent.lru.Remove(entry)
		agent.metrics.event_horizon = agent.now.Sub(trace.last_modified)

		// Manage buffers
		evicted_buffers = append(evicted_buffers, trace.buffers...)
	}

	// Make the evicted buffers available
	if len(evicted_buffers) > 0 {
		agent.cache_size -= len(evicted_buffers)
		agent.api.Available <- evicted_buffers
	}
}

func (agent *Agent) addBuffersToTrace(trace *TraceData, buffers []int) {
	if len(trace.triggered_by) > 0 {
		/* The trace is triggered and got some new buffers.
		It's not currently slated for reporting, but now needs to be */
		for _, ts := range trace.triggered_by {
			if !trace.needs_reporting {
				ts.trigger.lru.Remove(ts.e)
				agent.triggered_lru_size -= 1
				ts.e = nil
				ts.trigger.unreported.Insert(trace.trace_id)
			}
			ts.trigger.size += len(buffers)
		}
		agent.triggered_size += len(buffers)
	} else {
		agent.lru.MoveToFront(trace.e)
		trace.last_modified = agent.now
	}

	trace.needs_reporting = true
	trace.buffers = append(trace.buffers, buffers...)
	agent.cache_size += len(buffers)
}

/* Add some buffers to the cache */
func (agent *Agent) processCompletedBuffers(batch memory.CompleteBatch) {

	var freed_buffers []int

	for trace_id, buffers := range batch {
		/* Update agent metrics */
		agent.metrics.complete_buffers += len(buffers)

		/* Ignore trace ID 0 */
		if trace_id == 0 {
			freed_buffers = append(freed_buffers, buffers...)
			continue
		}

		if trace, ok := agent.data[trace_id]; ok {
			agent.addBuffersToTrace(trace, buffers)
		} else {
			/* Buffers for a trace not seen yet */
			var trace TraceData
			trace.trace_id = trace_id
			trace.e = agent.lru.PushFront(&trace)
			trace.last_modified = agent.now
			agent.data[trace_id] = &trace

			agent.addBuffersToTrace(&trace, buffers)
		}
	}

	/* Trigger eviction if above threshold */
	agent.maybeEvict()

	/* Send freed buffers */
	if len(freed_buffers) > 0 {
		agent.api.Available <- freed_buffers
	}

	agent.metrics.complete_batches++
}

func (agent *Agent) addBreadcrumbsToTrace(trace *TraceData, breadcrumbs []string) {
	if len(trace.triggered_by) > 0 {
		if !trace.needs_reporting {
			/* The trace is triggered and got some new breadcrumbs.
			It's not currently slated for reporting, but now needs to be */
			for _, ts := range trace.triggered_by {
				ts.trigger.lru.Remove(ts.e)
				agent.triggered_lru_size -= 1
				ts.e = nil
				ts.trigger.unreported.Insert(trace.trace_id)
			}
		}
	} else {
		/* Trace isn't triggered; update LRU */
		agent.lru.MoveToFront(trace.e)
		trace.last_modified = agent.now
	}
	trace.needs_reporting = true
	trace.breadcrumbs = append(trace.breadcrumbs, breadcrumbs...)
}

/* Add some breadcrumbs to the cache */
func (agent *Agent) processBreadcrumbs(batch memory.BreadcrumbBatch) {
	for trace_id, breadcrumbs := range batch {
		if trace, ok := agent.data[trace_id]; ok {
			agent.addBreadcrumbsToTrace(trace, breadcrumbs)
		} else {
			/* Breadcrumbs for a trace not seen yet */
			var trace TraceData
			trace.trace_id = trace_id
			trace.e = agent.lru.PushFront(&trace)
			agent.data[trace_id] = &trace

			agent.addBreadcrumbsToTrace(&trace, breadcrumbs)
		}
	}
}

func (agent *Agent) trigger(trace *TraceData, trigger *TriggerState) {
	/* Remove from the LRU if it's in it */
	if trace.e != nil {
		agent.lru.Remove(trace.e)
		trace.e = nil
	}

	/* The first time a trace is triggered, report it regardless of content */
	if len(trace.triggered_by) == 0 {
		trace.needs_reporting = true
		agent.triggered_size += len(trace.buffers)
	}

	/* Has this trigger already fired for this trace? */
	var tt *TriggeredTrace
	for _, candidate := range trace.triggered_by {
		if candidate.trigger == trigger {
			tt = candidate
			break
		}
	}

	if tt == nil {
		/* A new trigger for this trace */

		tt = &TriggeredTrace{}
		tt.trigger = trigger

		if trace.needs_reporting {
			tt.e = nil
			trigger.unreported.Insert(trace.trace_id)
			trigger.size += len(trace.buffers)
		} else {
			tt.e = tt.trigger.lru.PushFront(trace)
			agent.triggered_lru_size += 1
		}

		trace.triggered_by = append(trace.triggered_by, tt)
	}
}

/* Drop the specified trace.  Requires that the trace isn't
currently triggered */
func (agent *Agent) drop(trace *TraceData) []int {
	if len(trace.triggered_by) != 0 {
		fmt.Println("Warning - dropping a triggered trace")
	}

	if trace.e != nil {
		agent.lru.Remove(trace.e)
		trace.e = nil
	}

	delete(agent.data, trace.trace_id)

	agent.cache_size -= len(trace.buffers)

	return trace.buffers
}

/* Untrigger the specified trigger for a trace.  If there are no triggers
remaining for the trace, then drop the trace. */
func (agent *Agent) untriggered(trace *TraceData, trigger *TriggerState) []int {
	for i, ts := range trace.triggered_by {
		if ts.trigger == trigger {
			/* Remove the trigger from the trace */
			l := len(trace.triggered_by) - 1
			trace.triggered_by[i] = trace.triggered_by[l]
			trace.triggered_by = trace.triggered_by[:l]

			/* Update trigger state */
			if ts.e != nil {
				trigger.lru.Remove(ts.e)
				ts.e = nil
				agent.triggered_lru_size -= 1
			} else {
				trigger.size -= len(trace.buffers)
			}

			trigger.metrics.evicted_buffers += len(trace.buffers)

			break
		}
	}

	/* If this was the last trigger, drop the trace */
	if len(trace.triggered_by) == 0 {
		agent.triggered_size -= len(trace.buffers)
		return agent.drop(trace)
	} else {
		return nil
	}
}

func (agent *Agent) getReport(trace *TraceData) *TraceData {
	/* Extract the data to report */
	var report TraceData
	report.trace_id = trace.trace_id
	report.buffers = trace.buffers
	report.breadcrumbs = trace.breadcrumbs

	size := len(report.buffers)

	/* Update the trace state */
	trace.buffers = nil
	trace.breadcrumbs = nil
	trace.needs_reporting = false

	for _, tt := range trace.triggered_by {
		tt.e = tt.trigger.lru.PushFront(trace)
		agent.triggered_lru_size += 1
		tt.trigger.size -= size
	}

	agent.triggered_size -= size
	agent.cache_size -= size

	return &report
}

func (agent *Agent) nextTraceToReport() *TraceData {
	for {
		var trigger *TriggerState
		for _, candidate := range agent.triggered {
			if candidate.unreported.Size() == 0 {
				candidate.vt = agent.vc // Catch up clock
			} else {
				/* Apply rate limiting */
				if candidate.limiter != nil && candidate.limiter.Available() < 0 {
					candidate.vt = agent.vc // Catch up clock
					continue
				}

				/* Apply fair sharing */
				if trigger == nil || candidate.vt < trigger.vt {
					trigger = candidate
				}
			}
		}

		if trigger == nil {
			return nil
		}

		// Report next trace for this trigger
		trace_id := trigger.unreported.PopMin()

		if trace, ok := agent.data[trace_id]; ok {
			if trace.needs_reporting {
				trigger.vt += len(trace.buffers)
				agent.vc = trigger.vt // Not quite correct; but enough for now

				/* Update rate limiter */
				if trigger.limiter != nil {
					trigger.limiter.Take(int64(len(trace.buffers) * agent.buffer_size))
				}

				trigger.metrics.reported_buffers += len(trace.buffers)

				return agent.getReport(trace)
			}
		}
	}
}

/* Sets a trace as triggered.  This is called by the client
invoking trigger over shm, and by the collector sending a
trigger to us.  We will send all current and future data
for this trace ID to the collector.  We will stop sending
data for this trace after 60 seconds.  */
func (agent *Agent) processTriggers(triggers []memory.Trigger) {
	for _, t := range triggers {
		trace_id := t.Request_id
		trigger := agent.getOrCreateTriggerState(t.Trigger_id)

		if trace, ok := agent.data[trace_id]; ok {
			/* Trigger for a trace already in cache */
			agent.trigger(trace, trigger)
		} else {
			/* Trigger for a trace not seen yet */
			var trace TraceData
			trace.trace_id = trace_id
			trace.e = nil
			agent.data[trace_id] = &trace

			agent.trigger(&trace, trigger)
		}

		trigger.metrics.count++
	}
}

func (agent *Agent) RunProcessingLoop(ctx context.Context) {
	fmt.Println("Agent goroutine running")
	var trace_to_report *TraceData
	for {
		agent.now = time.Now()

	Reporting:
		for {
			/* Attempt to send the report to the reporter.  The reporter
			has only a very short blocking queue, so this will often fail */
			if trace_to_report != nil {
				select {
				case agent.reporting.queue <- trace_to_report:
					trace_to_report = nil
				default:
					// Queue of pending reports is full
					break Reporting
				}
			}

			// Prepare the next report
			trace_to_report = agent.nextTraceToReport()

			if trace_to_report == nil {
				break Reporting
			}
		}

		select {
		case <-ctx.Done():
			fmt.Println("Agent goroutine exiting")
			return
		case triggers := <-agent.triggers:
			/* Received some triggers from the shm triggers queue */
			agent.processTriggers(triggers)
		case buffers := <-agent.api.Complete:
			/* Received some buffers from the shm complete queue */
			agent.processCompletedBuffers(buffers)
		case breadcrumbs := <-agent.api.Breadcrumbs:
			/* Received some breadcrumbs from the shm breadcrumbs queue */
			agent.processBreadcrumbs(breadcrumbs)
		default:
			/* Do nothing; try reporting again */
		}
	}
}

func (agent *Agent) Run(ctx context.Context) {
	wg := new(sync.WaitGroup)
	wg.Add(4)
	go func() {
		agent.RunProcessingLoop(ctx)
	}()
	go func() {
		agent.reporting.Run(ctx)
		wg.Done()
	}()
	go func() {
		agent.api.Run(ctx)
		wg.Done()
	}()
	go func() {
		agent.printLoop(ctx)
		wg.Done()
	}()
	wg.Wait()
}
