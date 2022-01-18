package agent

import (
	"container/list"
	"time"

	"github.com/geraldleizhang/hindsight/agent/pkg/util"
)

/*
The DataManager stores all trace and trigger data.  The DataManager
is not responsible for implementing eviction policies, timing traces out,
and so on - that is handled externally by the Agent
*/
type DataManager struct {
	now          time.Time
	traces       map[uint64]*Trace
	untriggered  UntriggeredData
	triggered    TriggeredData
	trace_count  int
	buffer_count int
}

type UntriggeredData struct {
	lru          *list.List // LRU of untriggered traces
	trace_count  int
	buffer_count int
}

type TriggeredData struct {
	trace_count  int
	buffer_count int
	queues       map[int]*TriggerQueue
}

type TriggerQueue struct {
	id           int
	trace_count  int
	buffer_count int
	fired        map[uint64]*FiredTrigger
	reporting    *util.TreeNode // queue for FiredTriggers with data to report
	idle         *list.List     // LRU for idle FiredTriggers
}

/* Called upon agent startup */
func InitDataManager() *DataManager {
	var dm DataManager
	dm.now = time.Now()
	dm.traces = make(map[uint64]*Trace)

	dm.triggered.buffer_count = 0
	dm.triggered.queues = make(map[int]*TriggerQueue)

	dm.untriggered.lru = list.New()
	dm.untriggered.buffer_count = 0

	return &dm
}

/* Buffers received from the shm queues */
func (dm *DataManager) AddBuffers(trace_id uint64, buffers []int) {
	trace := dm.getOrCreateTrace(trace_id)
	trace.AddBuffers(dm, buffers)
}

/* Breadcrumbs received from the shm queues */
func (dm *DataManager) AddBreadcrumbs(trace_id uint64, breadcrumbs []string) {
	trace := dm.getOrCreateTrace(trace_id)
	trace.AddBreadcrumbs(dm, breadcrumbs)
}

/* A trigger has fired. */
func (dm *DataManager) Trigger(queue_id int, trigger_id uint64, trace_ids []uint64) []string {
	trigger := dm.getOrCreateTrigger(queue_id, trigger_id)
	var breadcrumbs []string
	for _, trace_id := range trace_ids {
		trace := dm.getOrCreateTrace(trace_id)
		breadcrumbs = append(breadcrumbs, trigger.AddTrace(dm, trace)...)
	}
	return breadcrumbs
}

/* Evict the LRU untriggered trace; returns its buffers */
func (dm *DataManager) Evict() []int {
	if dm.untriggered.trace_count == 0 {
		return nil
	}

	trace := dm.untriggered.lru.Back().Value.(*Trace)
	return trace.TakeBuffers(dm)
}

/* Evicts multiple LRU untriggered traces to reach the target number of buffers.
Returns the evicted buffers to be freed */
func (dm *DataManager) EvictToCapacity(target_capacity int) []int {
	if dm.buffer_count <= target_capacity || target_capacity < 0 {
		return nil
	}

	/*
		We evict in batches for efficiency rather than one at a time; here
		calculate the number to actually evict, rounding up
	*/
	num_to_evict := dm.buffer_count - target_capacity
	min_to_evict := target_capacity / 100
	if num_to_evict < min_to_evict {
		num_to_evict = min_to_evict
	}

	/*
		Do the eviction
	*/
	var evicted []int
	for len(evicted) < num_to_evict && dm.untriggered.lru.Len() > 0 {
		trace := dm.untriggered.lru.Back().Value.(*Trace)
		evicted = append(evicted, trace.TakeBuffers(dm)...)
	}
	return evicted
}

func (dm *DataManager) getOrCreateTrace(trace_id uint64) *Trace {
	if trace, ok := dm.traces[trace_id]; ok {
		return trace
	} else {
		return initUntriggeredTrace(dm, trace_id)
	}
}

/*
We only create the trigger metadata the first time a trigger fires for a
queue ID.  A trigger is never destroyed, for now.

TODO: a buggy trigger could fire for random queueIds resulting in too many
queues being created.  A future fix would be to limit the number of
allowed empty queues and tear them down on an LRU basis.
*/
func (dm *DataManager) GetOrCreateQueue(queue_id int) *TriggerQueue {
	if queue, ok := dm.triggered.queues[queue_id]; ok {
		return queue
	}

	var queue TriggerQueue
	queue.id = queue_id
	queue.buffer_count = 0
	queue.fired = make(map[uint64]*FiredTrigger)
	queue.reporting = util.InitPartialPriorityTree()
	queue.idle = list.New()
	dm.triggered.queues[queue_id] = &queue
	return &queue
}

/*
When a trigger fires locally or remotely, we call this method to create
the metadata related to the trigger
*/
func (dm *DataManager) getOrCreateTrigger(queue_id int, id uint64) *FiredTrigger {
	queue := dm.GetOrCreateQueue(queue_id)

	if trigger, ok := queue.fired[id]; ok {
		return trigger
	} else {
		return initIdleTrigger(dm, id, queue)
	}
}

/* Evicts one fired trigger from the specified queue, and returns buffers to be freed */
func (dm *DataManager) EvictNext(queue *TriggerQueue) []int {
	id := queue.reporting.PopNearMax()
	trigger := queue.fired[id]
	return trigger.Evict(dm)
}

/* Pops one fired trigger from the specified queue, and returns buffers to be reported and freed */
func (dm *DataManager) ReportNext(queue *TriggerQueue) []int {
	id := queue.reporting.PopMin()
	trigger := queue.fired[id]
	return trigger.GetBuffersForReport(dm)
}

/* Evict triggers that have been idle since before the specified time.
Since they are idle, this should not return any buffers */
func (dm *DataManager) EvictIdleTriggers(ft *FiredTrigger, expiration time.Duration) {
	// TODO
}
