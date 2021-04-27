package queue

import (
	"testing"
)

func TestInt32AndBytes(t *testing.T) {
	var (
		in       = int32(1)
		expected = int32(1)
	)
	actual := BytesToInt32(Int32ToBytes(in))
	if actual != expected {
		t.Errorf("Int value = %d; expected %d", actual, expected)
	}
}

func TestInt64AndBytes(t *testing.T) {
	var (
		in       = int64(1)
		expected = int64(1)
	)
	actual := BytesToInt64(Int64ToBytes(in))
	if actual != expected {
		t.Errorf("Int value = %d; expected %d", actual, expected)
	}
}

func TestQueueInit(t *testing.T) {
	var (
		in_fname = "/dev/shm/queue_test"
		in_size  = 200
		expected = 1616
	)
	actual := len(QueueInit(in_fname, in_size).queue)
	if actual != expected {
		t.Errorf("Queue size = %d; expected %d", actual, expected)
	}
}

func TestQueueData(t *testing.T) {
	var (
		in_fname = "/dev/shm/queue_test"
		in_size  = 200
	)
	queue := QueueInit(in_fname, in_size)
	QueuePut(queue, 123)
	actual_head := get_val(queue, header_idx(HEAD))
	if actual_head != 1 {
		t.Errorf("Head = %d; expected %d", actual_head, 1)
	}
	actual_tail := get_val(queue, header_idx(TAIL))
	if actual_tail != 0 {
		t.Errorf("Tail = %d; expected %d", actual_tail, 0)
	}

	QueueGet(queue)
	actual_head = get_val(queue, header_idx(HEAD))
	if actual_head != 1 {
		t.Errorf("Head = %d; expected %d", actual_head, 1)
	}
	actual_tail = get_val(queue, header_idx(TAIL))
	if actual_tail != 1 {
		t.Errorf("Tail = %d; expected %d", actual_tail, 1)
	}
}
