package agent

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Uint64() uint64 {
	return uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
}

func TestDataManagerFromScratch(t *testing.T) {
	assert := assert.New(t)

	dm := InitDataManager()

	/*
		First, add a trace, and check it gets inserted correctly
		with the correct buffers and correct counts
	*/
	dm.AddBuffers(75, []int{3, 12})

	assert.Equal(dm.trace_count, 1, "Trace count")
	assert.Equal(dm.buffer_count, 2, "Buffer count")
	assert.Equal(dm.untriggered.trace_count, 1, "Untriggered trace count")
	assert.Equal(dm.untriggered.buffer_count, 2, "Untriggered buffer count")
	assert.Equal(dm.triggered.trace_count, 0, "Triggered trace count")
	assert.Equal(dm.triggered.buffer_count, 0, "Triggered buffer count")

	switch v := dm.traces[75].state.(type) {
	case untriggeredTrace:
		assert.Equal(v.buffers, []int{3, 12}, "Buffers are correct")
	default:
		assert.Fail("Unexpected trace type")
	}

	/*
		Add some more buffers to the trace, check it gets updated
	*/
	dm.AddBuffers(75, []int{55, 2})

	assert.Equal(dm.trace_count, 1, "Trace count")
	assert.Equal(dm.buffer_count, 4, "Buffer count")
	assert.Equal(dm.untriggered.trace_count, 1, "Untriggered trace count")
	assert.Equal(dm.untriggered.buffer_count, 4, "Untriggered buffer count")

	switch v := dm.traces[75].state.(type) {
	case untriggeredTrace:
		assert.Equal(v.buffers, []int{3, 12, 55, 2}, "Buffers are correct")
		assert.Equal(len(v.breadcrumbs), 0, "No breadcrumbs yet")
	default:
		assert.Fail("Unexpected trace type")
	}

	/*
		Add some breadcrumbs, check they get added too, and counts are correct
	*/
	dm.AddBreadcrumbs(75, []string{"hello", "world"})

	assert.Equal(dm.trace_count, 1, "Trace count")
	assert.Equal(dm.buffer_count, 4, "Buffer count")
	assert.Equal(dm.untriggered.trace_count, 1, "Untriggered trace count")
	assert.Equal(dm.untriggered.buffer_count, 4, "Untriggered buffer count")
	assert.Equal(dm.triggered.trace_count, 0, "Triggered trace count")
	assert.Equal(dm.triggered.buffer_count, 0, "Triggered buffer count")

	switch v := dm.traces[75].state.(type) {
	case untriggeredTrace:
		assert.Equal(v.buffers, []int{3, 12, 55, 2}, "Buffers are correct")
		assert.Equal(v.breadcrumbs, []string{"hello", "world"}, "Breadcrumbs are correct")
	default:
		assert.Fail("Unexpected trace type")
	}

	/*
		Add some other traces, check they get added and counts are correct
	*/
	dm.AddBuffers(25, []int{100, 101, 102, 103, 104})
	dm.AddBuffers(50, []int{200, 201, 202, 203, 204, 205, 206, 207})
	dm.AddBreadcrumbs(100, []string{"breadcrumbs", "only"})

	assert.Equal(dm.trace_count, 4, "Trace count")
	assert.Equal(dm.buffer_count, 17, "Buffer count")
	assert.Equal(dm.untriggered.trace_count, 4, "Untriggered trace count")
	assert.Equal(dm.untriggered.buffer_count, 17, "Untriggered buffer count")
	assert.Equal(dm.triggered.trace_count, 0, "Triggered trace count")
	assert.Equal(dm.triggered.buffer_count, 0, "Triggered buffer count")

	assert.Equal(len(dm.triggered.queues), 0, "Shouldn't have triggers yet")

	/*
		Transition to triggered, check that we get back the breadcrumbs immediately
		and that we transition to reporting
	*/
	breadcrumbs := dm.Trigger(1, 75, []uint64{75})

	assert.Equal(dm.trace_count, 4, "Trace count")
	assert.Equal(dm.buffer_count, 17, "Buffer count")
	assert.Equal(dm.untriggered.trace_count, 3, "Untriggered trace count")
	assert.Equal(dm.untriggered.buffer_count, 13, "Untriggered buffer count")
	assert.Equal(dm.triggered.trace_count, 1, "Triggered trace count")
	assert.Equal(dm.triggered.buffer_count, 4, "Triggered buffer count")

	assert.Equal(breadcrumbs, []string{"hello", "world"}, "Breadcrumbs were returned upon triggering")

	assert.Equal(len(dm.triggered.queues), 1, "Trigger queue was created")
	assert.NotNil(dm.triggered.queues[1], "Trigger queue was created")

	q := dm.triggered.queues[1]
	assert.Equal(q.id, 1, "Trigger queue ID created correctly")
	assert.Equal(q.trace_count, 1, "Trigger queue trace count")
	assert.Equal(q.buffer_count, 4, "Trigger queue buffer count")

	assert.Equal(len(q.fired), 1, "FiredTrigger exists")
	assert.NotNil(q.fired[75], "FiredTrigger exists")

	trigger := q.fired[75]
	assert.Equal(trigger.id, uint64(75), "FiredTrigger ID")
	assert.Equal(trigger.buffer_count, 4, "FiredTrigger buffer count")
	assert.Equal(trigger.queue, q, "FiredTrigger queue")
	assert.Equal(len(trigger.traces), 1, "Trace was added to FiredTrigger")
	assert.NotNil(trigger.traces[75], "Trace was added to FiredTrigger")

	switch v := dm.traces[75].state.(type) {
	case reportingTrace:
		assert.Equal(v.buffers, []int{3, 12, 55, 2}, "Buffers are correct")
		assert.Equal(len(v.triggers), 1, "Trigger was registered to the trace correctly")
	default:
		assert.Fail("Unexpected trace type, expected to be reportingTrace")
	}

	switch trigger.state.(type) {
	case reportingTrigger:
	default:
		assert.Fail("Unexpected trigger type, expected to be reportingTrigger")
	}
}

/*
Creates a data manager used by most tests.

Trace IDs:
 * [0, 10) have 1, 2, 3, 4, ... buffers respectively, are untriggered
 *
*/
func initDataManagerForTest() *DataManager {
	dm := InitDataManager()

	for i := 0; i < 10; i++ {
		var buffers []int
		for j := 0; j < i+1; j++ {
			buffers = append(buffers, 1000*(i+1)+j)
		}
		dm.AddBuffers(uint64(i), buffers)
	}

	// fmt.Println(dm)

	return dm
}

/*
Preconditions:
* Expect the datamanager to be prepopulated with 10 untriggered traces
*/
func TestUntriggeredLRU(t *testing.T) {
	assert := assert.New(t)

	dm := initDataManagerForTest()

	/*
		Preconditions: expect the DM to be prepopulated with 10 untriggered traces
	*/
	assert.Equal(dm.untriggered.trace_count, 10, "Untriggered trace count")

	/*
		Drain all untriggered traces
	*/
	trace_count := dm.trace_count
	buf_count := dm.buffer_count
	untriggered_buf_count := dm.untriggered.buffer_count
	triggered_buf_count := dm.triggered.buffer_count
	triggered_trace_count := dm.triggered.trace_count

	for i := 9; i >= 0; i-- {
		evicted := dm.Evict()

		untriggered_buf_count -= len(evicted)
		buf_count -= len(evicted)
		trace_count -= 1

		assert.Equal(trace_count, dm.trace_count, "Global trace count decremented")
		assert.Equal(i, dm.untriggered.trace_count, "Untriggered trace count decremented")
		assert.Equal(triggered_trace_count, dm.triggered.trace_count, "Triggered trace count remains unchanged")

		assert.Equal(buf_count, dm.buffer_count, "Global buffer count decremented")
		assert.Equal(untriggered_buf_count, dm.untriggered.buffer_count, "Untriggered buffer count decremented")
		assert.Equal(triggered_buf_count, dm.triggered.buffer_count, "Triggered buffer count remains unchanged")
	}

	assert.Equal(0, dm.untriggered.buffer_count, "No untriggered buffers remaining")
	assert.Equal(0, dm.untriggered.trace_count, "No untriggered traces remaining")
	assert.Equal(dm.buffer_count, dm.triggered.buffer_count, "Global buffer count is equal to triggered buffer count")
	assert.Equal(dm.trace_count, dm.triggered.trace_count, "Global trace count is equal to triggered trace count")
	assert.Equal(triggered_buf_count, dm.triggered.buffer_count, "Triggered buffer count remains unchanged")
	assert.Equal(triggered_trace_count, dm.triggered.trace_count, "Triggered trace count remains unchanged")

	/*
		Evict should return nil when nothing to evict and should leave
		triggered traces unaffected
	*/

	evicted := dm.Evict()
	assert.Nil(evicted, "Nothing evicted when nothing exists")

	assert.Equal(0, dm.untriggered.buffer_count, "No untriggered buffers remaining")
	assert.Equal(0, dm.untriggered.trace_count, "No untriggered traces remaining")
	assert.Equal(dm.buffer_count, dm.triggered.buffer_count, "Global buffer count is equal to triggered buffer count")
	assert.Equal(dm.trace_count, dm.triggered.trace_count, "Global trace count is equal to triggered trace count")
	assert.Equal(triggered_buf_count, dm.triggered.buffer_count, "Triggered buffer count remains unchanged")
	assert.Equal(triggered_trace_count, dm.triggered.trace_count, "Triggered trace count remains unchanged")

	/*
		Add some untriggered traces
	*/
	for i := 0; i < 20; i++ {
		dm.AddBuffers(uint64(i), []int{2 * i, 2*i + 1})
		assert.Equal(2*(i+1), dm.untriggered.buffer_count, "Untriggered buffers were added")
		assert.Equal(i+1, dm.untriggered.trace_count, "Untriggered traces were added")
	}

	/*
		Evict should evict in LRU order
	*/
	for i := 0; i < 10; i++ {
		evicted := dm.Evict()
		assert.Equal(2, len(evicted), "Evicted 2 buffers")
		assert.Equal(2*i, evicted[0], "Evicted the right buffers")
		assert.Equal(2*i+1, evicted[1], "Evicted the right buffers")
	}

	/*
		Adding buffers should update LRU
	*/
	for i := 19; i >= 10; i-- {
		dm.AddBuffers(uint64(i), []int{2*i + 2, 2*i + 3})
	}

	/*
		Evict should evict in LRU order
	*/
	for i := 19; i >= 10; i-- {
		evicted := dm.Evict()
		assert.Equal(4, len(evicted), "Evicted 4 buffers")
		assert.Equal(2*i, evicted[0], "Evicted the right buffers")
		assert.Equal(2*i+1, evicted[1], "Evicted the right buffers")
		assert.Equal(2*i+2, evicted[2], "Evicted the right buffers")
		assert.Equal(2*i+3, evicted[3], "Evicted the right buffers")
	}

	/*
		Shouldn't be any untriggered buffers or traces remaining
	*/
	assert.Equal(0, dm.untriggered.buffer_count, "No untriggered buffers remaining")
	assert.Equal(0, dm.untriggered.trace_count, "No untriggered traces remaining")
	assert.Equal(dm.buffer_count, dm.triggered.buffer_count, "Global buffer count is equal to triggered buffer count")
	assert.Equal(dm.trace_count, dm.triggered.trace_count, "Global trace count is equal to triggered trace count")
	assert.Equal(triggered_buf_count, dm.triggered.buffer_count, "Triggered buffer count remains unchanged")
	assert.Equal(triggered_trace_count, dm.triggered.trace_count, "Triggered trace count remains unchanged")

}

func TestTriggeredArentEvicted(t *testing.T) {
	assert := assert.New(t)

	dm := initDataManagerForTest()

	/*
		Preconditions: expect the DM to be prepopulated with 10 untriggered traces
		with trace IDs [0, 10)
	*/
	assert.Equal(dm.untriggered.trace_count, 10, "Untriggered trace count")

	trace_count_before := dm.trace_count
	buf_count_before := dm.buffer_count

	/*
		Trigger all untriggered traces
	*/
	for i := 0; i < 10; i++ {
		dm.Trigger(0, uint64(i), []uint64{uint64(i)})
	}

	assert.Equal(0, dm.untriggered.trace_count, "Expect no untriggered traces")
	assert.Equal(0, dm.untriggered.buffer_count, "Expect no untriggered traces")
	assert.Equal(trace_count_before, dm.trace_count, "Total traces remains unchanged")
	assert.Equal(buf_count_before, dm.buffer_count, "Total buffers remains unchanged")
	assert.Equal(dm.trace_count, dm.triggered.trace_count, "All traces are triggered")
	assert.Equal(dm.buffer_count, dm.triggered.buffer_count, "All buffers are triggered")

	/*
		Evict should return nil when nothing to evict and should leave
		triggered traces unaffected
	*/

	evicted := dm.Evict()
	assert.Nil(evicted, "Nothing evicted when nothing exists")

	assert.Equal(0, dm.untriggered.trace_count, "Expect no untriggered traces")
	assert.Equal(0, dm.untriggered.buffer_count, "Expect no untriggered traces")
	assert.Equal(trace_count_before, dm.trace_count, "Total traces remains unchanged")
	assert.Equal(buf_count_before, dm.buffer_count, "Total buffers remains unchanged")
	assert.Equal(dm.trace_count, dm.triggered.trace_count, "All traces are triggered")
	assert.Equal(dm.buffer_count, dm.triggered.buffer_count, "All buffers are triggered")
}
