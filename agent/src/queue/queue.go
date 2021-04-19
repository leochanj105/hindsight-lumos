package queue

import (
	"fmt"
	"os"
	"syscall"

	. "util"
)

const (
	HEAD  int = 0
	TAIL  int = 1
	CAP   int = 2
	COUNT int = 3

	CONST_NUM int = 4
	LEN       int = 4
)

type Queue struct {
	queue []byte
}

var Complete Queue
var Available Queue
var Triggers Queue

func header_idx(index int) int {
	return LEN * index
}

func val_idx(index int) int {
	return LEN*4 + index*8
}

func avl_idx(index int) int {
	return LEN*4 + index*8 + 4
}

func set_val(queue Queue, index int, value int) {
	copy(queue.queue[index:index+LEN], Int32ToBytes(int32(value)))
}

func get_val(queue Queue, index int) int {
	return int(BytesToInt32(queue.queue[index : index+LEN]))
}

func QueueInit(fname string, size int) Queue {
	for {
		if IsFileExists(fname) == true {
			break
		}
	}
	queue_size := CONST_NUM*LEN + size*LEN*2
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("open file failed:", err)
	}
	fd := int(f.Fd())
	syscall.Ftruncate(fd, int64(queue_size))

	queue, err := syscall.Mmap(fd, 0, queue_size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)

	if err != nil {
		fmt.Println("mmap failed:", err)
	}

	var queue_handle Queue
	queue_handle.queue = queue

	// for i := 0; i < queue_size; i++ {
	// 	queue[i] = 0
	// }

	// set_val(queue_handle, header_idx(HEAD), 0)
	// set_val(queue_handle, header_idx(TAIL), 0)
	// set_val(queue_handle, header_idx(CAP), size)
	// set_val(queue_handle, header_idx(COUNT), 0)

	return queue_handle
}

func get_head(queue Queue) int {
	cap := get_val(queue, header_idx(CAP))
	curr_head := get_val(queue, header_idx(HEAD))

	for {
		if get_val(queue, avl_idx(curr_head)) == 0 {
			break
		}
	}

	set_val(queue, avl_idx(curr_head), 1)
	set_val(queue, header_idx(HEAD), (curr_head+1)%cap)

	return curr_head
}

func QueuePut(queue Queue, data int) {
	head := get_head(queue)
	if DEBUG == 1 {
		fmt.Println("[queue_put] head:", head, " tail:", get_val(queue, header_idx(TAIL)))
	}

	set_val(queue, val_idx(head), data)
	set_val(queue, avl_idx(head), 2)
	count := get_val(queue, header_idx(COUNT))
	set_val(queue, header_idx(COUNT), count+1)
	return
}

func get_tail(queue Queue) int {
	cap := get_val(queue, header_idx(CAP))
	curr_tail := get_val(queue, header_idx(TAIL))

	counter := 0
	for {
		if get_val(queue, avl_idx(curr_tail)) == 2 {
			break
		}
		counter += 1
		if counter == 1000 {
			return -1
		}
	}

	set_val(queue, avl_idx(curr_tail), 3)
	set_val(queue, header_idx(TAIL), (curr_tail+1)%cap)

	return curr_tail
}

func QueueGet(queue Queue) int {
	tail := get_tail(queue)
	if tail == -1 {
		return -1
	}
	if DEBUG == 1 {
		fmt.Println("[queue_get] head:", get_val(queue, header_idx(HEAD)), " tail:", tail)
	}
	data := get_val(queue, val_idx(tail))
	set_val(queue, avl_idx(tail), 0)
	return data
}

func QueueInitTest() {
	fmt.Println(get_val(Complete, header_idx(CAP)))
	fmt.Println(get_val(Available, header_idx(CAP)))
	fmt.Println(get_val(Triggers, header_idx(CAP)))

	// fmt.Println(get_val(queue, header_idx(HEAD)))
	// fmt.Println(get_val(queue, header_idx(TAIL)))
	// fmt.Println(get_val(queue, header_idx(CAP)))
	// fmt.Println(get_val(queue, header_idx(COUNT)))
	// for i := 0; i < 10; i++ {
	// 	fmt.Println(get_val(queue, val_idx(i)), get_val(queue, avl_idx(i)))
	// 	fmt.Println(queue.queue[16+i*8 : 24+i*8])
	// }

	return
}

func AvailableInit(cap int) {
	for i := 0; i < cap; i++ {
		QueuePut(Available, i)
	}
}

func print_stat(queue Queue) {
	fmt.Println(get_val(queue, header_idx(HEAD)), get_val(queue, header_idx(TAIL)), get_val(queue, header_idx(CAP)), get_val(queue, header_idx(COUNT)))
}

func PrintQueueStat() {
	fmt.Println("[queue_stat] available:")
	print_stat(Available)
	fmt.Println("[queue_stat] complete:")
	print_stat(Complete)
}
