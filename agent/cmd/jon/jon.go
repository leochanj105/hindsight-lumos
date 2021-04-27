package main

// IMPORTANT: for the below to work, must do:
//   export CGO_LDFLAGS_ALLOW=".*"

/*
#cgo CFLAGS: -I${SRCDIR}/../../../client/src
#cgo LDFLAGS: ${SRCDIR}/../../../client/lib/libtracer.a

#include "agentapi.h"


*/
import "C"

import (
	"fmt"
	"time"
)

func reset_available_buffers(api *C.HindsightAgentAPI) {
	davailable := 0
	for {
		var ab C.AvailableBuffers
		C.hindsight_agentapi_get_available_nonblocking(api, &ab)

		if (ab.count == 0) {
			break
		}

		davailable += int(ab.count)
	}

	dcomplete := 0
	for {
		var cb C.CompleteBuffers
		C.hindsight_agentapi_get_complete_nonblocking(api, &cb)

		if (cb.count == 0) {
			break
		}

		dcomplete += int(cb.count)
	}

	fmt.Println("Resetting buffers: drained", davailable, "available and", dcomplete, "complete")
}

func make_all_buffers_available(api *C.HindsightAgentAPI) {
	fmt.Println("Initialize buffers: making", api.mgr.meta.capacity, "buffers available...")

	next_buffer_id := 0
	remaining := api.mgr.meta.capacity
	for (remaining > 0) {
		var av C.AvailableBuffers
		av.count = 100
		if (av.count > remaining) {
			av.count = remaining
		}
		remaining -= av.count

		for i:= 0; i < 100; i++ {
			av.bufs[i].buffer_id = C.int(next_buffer_id)
			next_buffer_id++
		}

		C.hindsight_agentapi_put_available_blocking(api, &av);
	}

	fmt.Println("Initialize buffers: done")
	fmt.Println("Queue states:")
	fmt.Print("  Available ")
	C.queue_print(&api.mgr.available)
	fmt.Print("  Complete ")
	C.queue_print(&api.mgr.complete)
}

func init_agentapi(fname string) *C.HindsightAgentAPI {
	agentapi := C.hindsight_agentapi_init(C.CString(fname))
	fmt.Println("Inited existing bufmanager", fname)
	reset_available_buffers(agentapi)
	make_all_buffers_available(agentapi);
	return agentapi
}


func drain_forever(api *C.HindsightAgentAPI) {
	last_print := int(time.Now().UnixNano())
	print_every := 1000000000
	count := 0
	sum := 0

	var cb C.CompleteBuffers
	for {
		now := int(time.Now().UnixNano())
		if ((now - last_print) > print_every) {
			tput := (sum * print_every) / (now - last_print)
			batchsize := float32(sum) / float32(count)
			fmt.Println("Throughput:", tput, "Average batch:", batchsize)
			last_print = now
			count = 0
			sum = 0
		}

		max_backoff := 100000
		backoff := int(10)
		for {
			C.hindsight_agentapi_get_complete_nonblocking(api, &cb)
			if (int(cb.count) > 0) {
				break
			}

			time.Sleep(time.Duration(backoff) * time.Nanosecond)
			backoff *= 2
			if (backoff > max_backoff) {
				backoff = max_backoff
			}
		}

		count += 1
		sum += int(cb.count)

		var ab C.AvailableBuffers
		ab.count = cb.count

		limit := int(cb.count)
		for i := 1; i < limit; i++ {
			ab.bufs[i].buffer_id = cb.bufs[i].buffer_id
		}

		C.hindsight_agentapi_put_available_blocking(api, &ab)
	}
}

func main() {
	fmt.Println("Hello world!")

	fname := "hs_integration_test"

	// f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0666)
	// if err != nil {
	// 	fmt.Println("open file failed:", err)
	// }
	// fd := int(f.Fd())
	// fmt.Println("opened ", fd)

	// fi, err := f.Stat()
	// if err != nil {
	// 	fmt.Println("stat failed:",err)
	// }
	// fmt.Println("size is ", fi.Size())

	// p, err := syscall.Mmap(fd, 0, int(fi.Size()), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)

	// fmt.Printf("%T\n", p)

	agentapi := init_agentapi(fname)
	drain_forever(agentapi)



	// fmt.Println(q)

	// fmt.Println(q.meta)

	// md := C.QueueMetadata(p)
}
