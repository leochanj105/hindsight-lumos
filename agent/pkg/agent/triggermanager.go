package agent

import "github.com/juju/ratelimit"

/*
This file defines the TriggerManager, which wraps the trigger queues of
the DataManager and applies rate limiting of triggers and of reporting.
*/
type TriggerManager struct {
	dm     *DataManager
	queues map[int]*ManagedQueue
	vc     int // Virtual clock used for fair sharing reporting across queues

	trigger_limit   float64 // Default limit an individual queue can trigger per second
	reporting_limit float64 // Default limit an individual queue can report per second
}

/*
Wraps a TriggerQueue in the DataManager
*/
type ManagedQueue struct {
	tm                *TriggerManager
	queue             *TriggerQueue     // The actual DataManager queue
	trigger_limiter   *ratelimit.Bucket // Rate limiter for local triggers
	reporting_limiter *ratelimit.Bucket // Rate limiter for reporting
	vt                int               // Virtual time used for fair sharing of reporting
	metrics           TriggerMetrics
}

func (tm *TriggerManager) Init(dm *DataManager) {
	tm.dm = dm
	tm.queues = make(map[int]*ManagedQueue)
	tm.vc = 0
	tm.trigger_limit = 10000                     // TODO: not hardcoded, configured per trigger, or adaptive based on eviction rates
	tm.reporting_limit = 10 * 1024 * 1024 * 1024 // TODO: not hardcoded
}

/*
Set per-trigger rate limits, configured via command line / config parameters

per_trigger_rate_limits is specified in MB/s
*/
func (tm *TriggerManager) ConfigureRateLimits(per_trigger_rate_limits map[int]float64) {
	for queue_id, limit := range per_trigger_rate_limits {
		queue := tm.getQueue(queue_id)
		limit_bytes := limit * 1024 * 1024
		queue.reporting_limiter = ratelimit.NewBucketWithRate(limit_bytes, int64(limit_bytes))
	}
}

func (tm *TriggerManager) getQueue(queue_id int) *ManagedQueue {
	if queue, ok := tm.queues[queue_id]; ok {
		return queue
	}

	var mq ManagedQueue
	mq.tm = tm
	mq.queue = tm.dm.GetQueue(queue_id)
	mq.trigger_limiter = ratelimit.NewBucketWithRate(tm.trigger_limit, int64(tm.trigger_limit))
	mq.reporting_limiter = ratelimit.NewBucketWithRate(tm.reporting_limit, int64(tm.reporting_limit))
	mq.vt = tm.vc

	tm.queues[queue_id] = &mq
	return &mq
}
