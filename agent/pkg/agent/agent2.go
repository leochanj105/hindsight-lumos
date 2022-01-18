package agent

import (
	"fmt"
	"time"

	"github.com/geraldleizhang/hindsight/agent/pkg/memory"
)

type Agent2 struct {
	dm        DataManager       // the trace data
	api       memory.GoAgentAPI // API to the shared memory
	reporting Reporting         // Interface to LogCollector
	tm        TriggerManager    // Rate limits and fair shares the triggers

	// Constants for deciding when to evict
	cache_capacity      int // Above this threshold, we should evict
	triggered_capacity  int // Above this threshold we evict from triggered
	buffer_size         int // Size of buffers in the cache
	eviction_batch_size int // Each eviction should aim for this many buffers

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
	agent.tm.Init(&agent.dm)
	agent.tm.ConfigureRateLimits(per_trigger_rate_limits)

	agent.cache_capacity = (4 * agent.api.Capacity()) / 5 // TODO: not hardcoded
	agent.triggered_capacity = agent.cache_capacity / 2   // TODO: not hardcoded
	agent.buffer_size = agent.api.BufferSize()
	agent.eviction_batch_size = agent.cache_capacity / 100 // Hard code to some value for now

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
