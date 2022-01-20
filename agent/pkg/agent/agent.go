package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/geraldleizhang/hindsight/agent/pkg/memory"
)

type Agent struct {
	dm          DataManager       // the trace data
	api         memory.GoAgentAPI // API to the shared memory
	coordinator Coordinator       // Interface to coordinator
	reporting   Reporting         // Interface to trace data backend
	tm          TriggerManager    // Rate limits and fair shares the triggers

	// Constants for deciding when to evict
	cache_capacity     int           // Above this threshold, we should evict
	triggered_capacity int           // Above this threshold we evict from triggered
	trigger_timeout    time.Duration // How long a trigger remains idle before being deleted

	/* Wraps api.Triggers, possibly adding a delay for experiments */
	localtriggers <-chan []memory.Trigger // triggers from shm

	metrics AgentMetrics

	vc int // Virtual clock used for fair sharing
}

func InitAgent2(fname string, trigger_delay uint64, reporting_rate_limit float64, trigger_rate_limit float64, per_trigger_rate_limits map[int]float64) *Agent {
	fmt.Println("Init agent", fname)
	fmt.Printf("  Trigger delay %d nanoseconds\n", trigger_delay)
	fmt.Printf("  Reporting rate limit %.2f MB/s\n", reporting_rate_limit)
	for trigger_id, rate := range per_trigger_rate_limits {
		fmt.Printf("    -Trigger %d rate limit %.2f MB/s\n", trigger_id, rate)
	}

	var agent Agent
	agent.dm.Init()
	agent.api.Init(fname)
	agent.reporting.Init(&agent.api, reporting_rate_limit, true)
	agent.coordinator.Init(true)
	agent.tm.Init(&agent.dm, agent.api.BufferSize(), trigger_rate_limit)
	agent.tm.ConfigureRateLimits(per_trigger_rate_limits)

	agent.cache_capacity = (4 * agent.api.Capacity()) / 5 // TODO: not hardcoded
	agent.triggered_capacity = agent.cache_capacity / 2   // TODO: not hardcoded
	agent.trigger_timeout = time.Duration(-5) * time.Minute

	// This delayed trigger stuff is only used for experiments with intentional trigger delay
	if trigger_delay == 0 {
		agent.localtriggers = agent.api.Triggers
	} else {
		proxy := make(chan []memory.Trigger, 10000)
		agent.localtriggers = proxy
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
func (agent *Agent) maybeEvict() {
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
func (agent *Agent) processBreadcrumbs(batch memory.BreadcrumbBatch) {
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

func (agent *Agent) processTriggers(batch []memory.Trigger) {
	triggers_to_forward := make([]memory.Trigger, 0, len(batch))
	breadcrumbs_to_forward := make(map[uint64][]string)
	for _, t := range batch {
		/* Add to the DataManager */
		// TODO: update C struct to send lateral trace ids all in one or have two ids
		queue := agent.tm.getQueue(t.Queue_id)
		triggered, breadcrumbs := queue.TriggerLocal(t.Base_trace_id, []uint64{t.Trace_id})

		/* Forward trigger to coordinator */
		if triggered {
			triggers_to_forward = append(triggers_to_forward, t)
		}

		/* Accumulate breadcrumbs to forward */
		for trace_id, addrs := range breadcrumbs {
			if len(addrs) > 0 {
				breadcrumbs_to_forward[trace_id] = append(breadcrumbs_to_forward[trace_id], addrs...)
			}
		}
	}

	/* Forward triggers and breadcrumbs */
	if len(triggers_to_forward) > 0 {
		select {
		case agent.coordinator.localtriggers <- triggers_to_forward:
			break
		default:
			// Connection to coordinator is bottlenecked; drop the triggers
		}
	}
	if len(breadcrumbs_to_forward) > 0 {
		select {
		case agent.coordinator.breadcrumbs <- breadcrumbs_to_forward:
			break
		default:
			// Connection to coordinator is bottlenecked; drop the breadcrumbs
		}
	}
}

func (agent *Agent) processRemoteTriggers(batch []memory.Trigger) {
	breadcrumbs_to_forward := make(map[uint64][]string)
	for _, t := range batch {
		queue := agent.tm.getQueue(t.Queue_id)
		// TODO: update C struct to send lateral trace ids all in one or have two ids
		breadcrumbs := queue.TriggerRemote(t.Base_trace_id, []uint64{t.Trace_id})

		/* Accumulate breadcrumbs to forward */
		for trace_id, addrs := range breadcrumbs {
			if len(addrs) > 0 {
				breadcrumbs_to_forward[trace_id] = append(breadcrumbs_to_forward[trace_id], addrs...)
			}
		}
	}

	if len(breadcrumbs_to_forward) > 0 {
		select {
		case agent.coordinator.breadcrumbs <- breadcrumbs_to_forward:
			break
		default:
			// Connection to coordinator is bottlenecked; drop the breadcrumbs
		}
	}
}

func (agent *Agent) RunProcessingLoop(ctx context.Context) {
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
				case agent.reporting.data <- data_to_report:
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
		case triggers := <-agent.coordinator.remotetriggers:
			/* Received some triggers from the coordinator */
			agent.processRemoteTriggers(triggers)
		case triggers := <-agent.localtriggers:
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

func (agent *Agent) Run(ctx context.Context) {
	wg := new(sync.WaitGroup)
	wg.Add(5)
	go func() {
		agent.RunProcessingLoop(ctx)
	}()
	go func() {
		agent.coordinator.Run(ctx)
		wg.Done()
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
