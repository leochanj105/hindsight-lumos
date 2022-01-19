package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/geraldleizhang/hindsight/agent/pkg/memory"
)

type Agent2 struct {
	dm        DataManager       // the trace data
	api       memory.GoAgentAPI // API to the shared memory
	reporting Reporting         // Interface to LogCollector
	tm        TriggerManager    // Rate limits and fair shares the triggers

	// Constants for deciding when to evict
	cache_capacity     int           // Above this threshold, we should evict
	triggered_capacity int           // Above this threshold we evict from triggered
	trigger_timeout    time.Duration // How long a trigger remains idle before being deleted

	/* Wraps api.Triggers, possibly adding a delay for experiments */
	triggers <-chan []memory.Trigger // triggers from shm

	metrics AgentMetrics

	vc int // Virtual clock used for fair sharing
}

func InitAgent2(fname string, trigger_delay uint64, rate_limit float64, per_trigger_rate_limits map[int]float64) *Agent2 {
	fmt.Println("Init agent", fname)
	fmt.Printf("  Trigger delay %d nanoseconds\n", trigger_delay)
	fmt.Printf("  Reporting rate limit %.2f MB/s\n", rate_limit)
	for trigger_id, rate := range per_trigger_rate_limits {
		fmt.Printf("    -Trigger %d rate limit %.2f MB/s\n", trigger_id, rate)
	}

	var agent Agent2
	agent.dm.Init()
	agent.api.Init(fname)
	agent.reporting.Init(&agent.api, rate_limit)
	agent.tm.Init(&agent.dm, agent.api.BufferSize())
	agent.tm.ConfigureRateLimits(per_trigger_rate_limits)

	agent.cache_capacity = (4 * agent.api.Capacity()) / 5 // TODO: not hardcoded
	agent.triggered_capacity = agent.cache_capacity / 2   // TODO: not hardcoded
	agent.trigger_timeout = time.Duration(-5) * time.Minute

	// This delayed trigger stuff is only used for experiments with intentional trigger delay
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

	fmt.Println("Go Agent cache capacity", agent.cache_capacity)

	return &agent
}

/*
	Checks the capacity of the DataManager and evicts buffers if necessary as follows:
	* Evict untriggered traces if above global capacity threshold
	* Evict triggers if above triggered buffer capacity threshold
	* Time out idle triggers
	Returns any evicted buffers to the available queue
*/
func (agent *Agent2) maybeEvict() {
	// Skip until the cache is full
	if agent.dm.buffer_count < agent.cache_capacity {
		return
	}

	// Clean up timed-out triggers
	agent.dm.CheckIdleTriggers(agent.dm.now.Add(agent.trigger_timeout))

	// Evict spammy triggers
	evicted := agent.dm.EvictedTriggeredToCapacity(agent.triggered_capacity)
	if len(evicted) > 0 {
		agent.api.Available <- evicted
	}

	// Evict untriggered trace data
	evicted = agent.dm.EvictToCapacity(agent.cache_capacity)
	if len(evicted) > 0 {
		agent.api.Available <- evicted
	}
	agent.metrics.event_horizon = agent.dm.now.Sub(agent.dm.untriggered.event_horizion)
}

/*
  Process a batch of buffers retrieved from the complete queue.
	This mainly just sends the buffers to the datamanager
*/
func (agent *Agent2) processCompletedBuffers(batch memory.CompleteBatch) {
	var freed_buffers []int
	for trace_id, buffers := range batch {
		/* Update agent metrics */
		agent.metrics.complete_buffers += len(buffers)

		/* Ignore trace ID 0 */
		if trace_id == 0 {
			freed_buffers = append(freed_buffers, buffers...)
			continue
		}

		/* Add to the DataManager */
		agent.dm.AddBuffers(trace_id, buffers)
	}

	/* Trigger eviction if necessary */
	agent.maybeEvict()

	/* Send freed buffers */
	if len(freed_buffers) > 0 {
		agent.api.Available <- freed_buffers
	}

	agent.metrics.complete_batches++
}

/*
  Process a batch of breadcrumbs retrieved from the shm bc queue.
	This mainly just sends the breadcrumbs to the datamanager
*/
func (agent *Agent2) processBreadcrumbs(batch memory.BreadcrumbBatch) {
	to_report := make(map[uint64][]string)
	for trace_id, breadcrumbs := range batch {
		/* Ignore trace ID 0 */
		if trace_id == 0 {
			continue
		}

		/* Add to the DataManager */
		breadcrumbs := agent.dm.AddBreadcrumbs(trace_id, breadcrumbs)
		if len(breadcrumbs) > 0 {
			to_report[trace_id] = breadcrumbs
		}
	}

	/* Forward breadcrumbs as needed */
	// TODO HERE
	// FORWARD TO COORDINATOR
	if len(to_report) > 0 {
		fmt.Printf("Forwarding crumbs %v\n", to_report)
	}
}

func (agent *Agent2) processTriggers(batch []memory.Trigger) {
	for _, t := range batch {

		// TODO: rate limiting goes here

		/* Add to the DataManager */
		// TODO: update C struct to send lateral trace ids all in one or have two ids
		queue := agent.tm.getQueue(t.Queue_id)
		breadcrumbs := queue.TriggerLocal(t.Base_trace_id, []uint64{t.Trace_id})

		/* Forward breadcrumbs as needed */
		// TODO HERE
		// FORWARD TO COORDINATOR
		if len(breadcrumbs) > 0 {
			fmt.Printf("Forwarding tcrumbs %v\n", breadcrumbs)
		}
	}
}

func (agent *Agent2) processRemoteTriggers(triggers map[TriggerID][]uint64) {
	for trigger_id, trace_ids := range triggers {
		queue := agent.tm.getQueue(trigger_id.queue_id)
		breadcrumbs := queue.TriggerRemote(trigger_id.base_trace_id, trace_ids)

		/* Forward breadcrumbs as needed */
		// TODO HERE
		// FORWARD TO COORDINATOR
		if breadcrumbs != nil {
			fmt.Printf("Forwarding tcrumbs %v\n", breadcrumbs)
		}
	}
}

func (agent *Agent2) RunProcessingLoop(ctx context.Context) {
	fmt.Println("Agent goroutine running")
	var data_to_report []int
	for {
		agent.dm.now = time.Now()

	Reporting:
		for {
			/* Attempt to send the report to the reporter.  The reporter
			has only a very short blocking queue, so this will often fail */
			if len(data_to_report) > 0 {
				select {
				case agent.reporting.queue <- data_to_report:
					data_to_report = nil
				default:
					break Reporting // Queue of pending reports is full
				}
			}

			// Prepare the next report
			data_to_report = agent.tm.GetNextBatchToReport()
			if len(data_to_report) == 0 {
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
			// TODO: separate channel for remote triggers
		}
	}
}

func (agent *Agent2) Run(ctx context.Context) {
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
