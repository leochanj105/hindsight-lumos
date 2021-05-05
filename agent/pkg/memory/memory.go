package memory

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
	"sync"
    "context"
    "time"
    "unsafe"
)

// BATCHSIZE is #defined in agentapi.h
//   The value here must be the same as the value in agentapi.h
const BATCHSIZE = 100

/* For directly putting and getting stuff from shm */
type AgentAPI struct {
	c_api *C.HindsightAgentAPI
}

type CompleteBatch map[uint64][]int
type BreadcrumbBatch map[uint64][]string

/* Go style API that has some goroutines and puts stuff into channels */
type GoAgentAPI struct {
	agent *AgentAPI

    Available chan []int    // Channel for re-enqueueing buffers to shm available queue
    Complete chan CompleteBatch // Channel for receiving completed buffers from shm
    Triggers chan []Trigger // Channel for receiving local triggers from shm
    Breadcrumbs chan BreadcrumbBatch // Channel for receiving breadcrumbs from shm
}

type CompleteBuffer struct {
	Request_id uint64
	Buffer_id int
}

type Trigger struct {
	Request_id uint64
	Trigger_id int
}

type Breadcrumb struct {
	Request_id uint64
	Address string
}

func InitAgentAPI(fname string) *AgentAPI {
	var agent AgentAPI
    agent.c_api = C.hindsight_agentapi_init(C.CString(fname))
    
    fmt.Println("Initialize buffers: done")
    fmt.Println("Queue states:")
    fmt.Print("  Available ")
    C.queue_print(&agent.c_api.mgr.available)
    fmt.Print("  Complete ")
    C.queue_print(&agent.c_api.mgr.complete)
    return &agent
}

func InitGoAgentAPI(fname string) *GoAgentAPI {
	var api GoAgentAPI
	api.agent = InitAgentAPI(fname)
	api.Available = make(chan []int, 100000)
	api.Complete = make(chan CompleteBatch, 100000)
	api.Triggers = make(chan []Trigger, 100000)
	api.Breadcrumbs = make(chan BreadcrumbBatch, 100000)
	return &api
}

func (api *GoAgentAPI) Capacity() int {
	return int(api.agent.c_api.mgr.meta.capacity)
}

func (api *GoAgentAPI) BufferSize() int {
	return int(api.agent.c_api.mgr.meta.buffer_size)
}

func (api *GoAgentAPI) Run(ctx context.Context) {
    fmt.Println("shm queue goroutine running")
    wg := new(sync.WaitGroup)
    wg.Add(4)
    go func() {
        api.availableLoop(ctx)
        wg.Done()
    }()
    go func() {
        api.completeLoop(ctx)
        wg.Done()
    }()
    go func() {
        api.triggerLoop(ctx)
        wg.Done()
    }()
    go func() {
        api.breadcrumbsLoop(ctx)
        wg.Done()
    }()
    wg.Wait()
}

func (api *GoAgentAPI) availableLoop(ctx context.Context) {
	for {
		select {
		case bufids := <- api.Available:
			api.agent.PutAvailable(bufids)
		case <- ctx.Done():
			return
		}
	}
}

func (api *GoAgentAPI) drainBatches(ctx context.Context, min_bs int) int {
	var total int
	total = 0
	for {
		select {
		case <- ctx.Done():
			return 0
		default:
			count, completed := api.agent.GetCompleteBatches()

			total += count
			if count > 0 {
				api.Complete <- completed
			}
			if count < min_bs {
				return total
			}
		}
	}
}

func (api *GoAgentAPI) completeLoop(ctx context.Context) {
	fmt.Println("completeLoop")
    max_backoff := 100000
    min_backoff := 10
    backoff := int(10)
    min_bs := 20
	for {
		// Keep processing batches so long as they are BATCHSIZE/2 large
		total := api.drainBatches(ctx, min_bs)

		if total < min_bs {
			// Back off exponentially
			backoff *= 2
		} else {
			backoff = min_backoff
		}

		// Keep within bounds
        if (backoff > max_backoff) {
            backoff = max_backoff
        }
		time.Sleep(time.Duration(backoff) * time.Nanosecond)
	}
}

func (api *GoAgentAPI) drainTriggers(ctx context.Context, min_bs int) int {
	var total int
	total = 0
	for {
		select {
		case <- ctx.Done():
			return 0
		default:
			triggers := api.agent.GetTriggers()

			count := len(triggers)
			total += count
			if count > 0 {
				api.Triggers <- triggers
			}
			if count < min_bs {
				return total
			}
		}
	}
}

func (api *GoAgentAPI) triggerLoop(ctx context.Context) {
    max_backoff := 100000
    min_backoff := 10
    backoff := int(10)
    min_bs := 20
	for {
		// Keep processing batches so long as they are BATCHSIZE/2 large
		total := api.drainTriggers(ctx, min_bs)

		if total < min_bs {
			// Back off exponentially
			backoff *= 2
		} else {
			backoff = min_backoff
		}

		// Keep within bounds
        if (backoff > max_backoff) {
            backoff = max_backoff
        }
		time.Sleep(time.Duration(backoff) * time.Nanosecond)
	}
}

func (api *GoAgentAPI) drainBreadcrumbs(ctx context.Context, min_bs int) int {
	var total int
	total = 0
	for {
		select {
		case <- ctx.Done():
			return 0
		default:
			count, breadcrumbs := api.agent.GetBreadcrumbBatches()

			total += count
			if count > 0 {
				api.Breadcrumbs <- breadcrumbs
			}
			if count < min_bs {
				return total
			}
		}
	}
}

func (api *GoAgentAPI) breadcrumbsLoop(ctx context.Context) {
    max_backoff := 100000
    min_backoff := 10
    backoff := int(10)
    min_bs := 20
	for {
		// Keep processing batches so long as they are BATCHSIZE/2 large
		total := api.drainBreadcrumbs(ctx, min_bs)

		if total < min_bs {
			// Back off exponentially
			backoff *= 2
		} else {
			backoff = min_backoff
		}

		// Keep within bounds
        if (backoff > max_backoff) {
            backoff = max_backoff
        }
		time.Sleep(time.Duration(backoff) * time.Nanosecond)
	}
}

/* Retrieves up to BATCHSIZE buffers from the complete queue.

BATCHSIZE is hard-coded in agentapi.h

This is a non-blocking call; may return 0 buffers
*/
func (agent *AgentAPI) GetComplete() []CompleteBuffer {
	var cb C.CompleteBuffers
	C.hindsight_agentapi_get_complete_nonblocking(agent.c_api, &cb)

	count := int(cb.count)
	buffers := make([]CompleteBuffer, count)
	for i := 0; i < count; i++ {
		buffer := &buffers[i]
		buffer.Request_id = uint64(cb.bufs[i].trace_id)
		buffer.Buffer_id = int(cb.bufs[i].buffer_id)
	}

	return buffers
}

/* Retrieves up to BATCHSIZE buffers from the complete queue.

Groups bufids by trace ID

BATCHSIZE is hard-coded in agentapi.h

This is a non-blocking call; may return 0 buffers
*/
func (agent *AgentAPI) GetCompleteBatches() (int, CompleteBatch) {
	var cb C.CompleteBuffers
	C.hindsight_agentapi_get_complete_nonblocking(agent.c_api, &cb)

	count := int(cb.count)
	buffers := make(CompleteBatch, count)
	for i := 0; i < count; i++ {
		trace_id := uint64(cb.bufs[i].trace_id)
		buffer_id := int(cb.bufs[i].buffer_id)
		buffers[trace_id] = append(buffers[trace_id], buffer_id)
	}

	return count, buffers
}

/* Puts buffers to the available queue.

This is a blocking call; it will wait until all available IDs
have been enqueued.

In practice this should never block if the queue capacity 
is equal to, or exceeds, the buffer pool capacity
*/
func (agent *AgentAPI) PutAvailable(ids []int) {
	var ab C.AvailableBuffers

	for len(ids) > 0 {
		size := len(ids)
		if size > BATCHSIZE {
			size = BATCHSIZE
		}

		ab.count = C.ulong(size)
		for i := 0; i < size; i++ {
			ab.bufs[i].buffer_id = C.int(ids[i])
		}

        C.hindsight_agentapi_put_available_blocking(agent.c_api, &ab);

		ids = ids[size:]
	}
}


/* Retrieves up to BATCHSIZE triggers from the triggers queue.

BATCHSIZE is hard-coded in agentapi.h

This is a non-blocking call; may return 0 triggers
*/
func (agent *AgentAPI) GetTriggers() []Trigger {
	var tb C.TriggerBatch
	C.hindsight_agentapi_get_triggers_nonblocking(agent.c_api, &tb)

	count := int(tb.count)
	triggers := make([]Trigger, count)
	for i := 0; i < count; i++ {
		trigger := &triggers[i]
		trigger.Request_id = uint64(tb.triggers[i].trace_id)
		trigger.Trigger_id = int(tb.triggers[i].trigger_id)
	}

	return triggers
}


/* Retrieves up to BATCHSIZE breadcrumbs from the breadcrumbs queue.

BATCHSIZE is hard-coded in agentapi.h

This is a non-blocking call; may return 0 breadcrumbs
*/
func (agent *AgentAPI) GetBreadcrumbs() []Breadcrumb {
	var bb C.BreadcrumbBatch
	C.hindsight_agentapi_get_breadcrumbs_nonblocking(agent.c_api, &bb)

	count := int(bb.count)
	breadcrumbs := make([]Breadcrumb, count)
	for i := 0; i < count; i++ {
		breadcrumb := &breadcrumbs[i]
		breadcrumb.Request_id = uint64(bb.breadcrumbs[i].trace_id)
		breadcrumb.Address = C.GoString(bb.breadcrumb_addrs[i])
	}

	return breadcrumbs	
}


/* Retrieves up to BATCHSIZE breadcrumbs from the breadcrumbs queue.

Groups breadcrumbs by trace ID

BATCHSIZE is hard-coded in agentapi.h

This is a non-blocking call; may return 0 breadcrumbs
*/
func (agent *AgentAPI) GetBreadcrumbBatches() (int, BreadcrumbBatch) {
	var bb C.BreadcrumbBatch
	C.hindsight_agentapi_get_breadcrumbs_nonblocking(agent.c_api, &bb)

	count := int(bb.count)
	breadcrumbs := make(BreadcrumbBatch, count)
	for i := 0; i < count; i++ {
		trace_id := uint64(bb.breadcrumbs[i].trace_id)
		addr := C.GoString(bb.breadcrumb_addrs[i])
		breadcrumbs[trace_id] = append(breadcrumbs[trace_id], addr)
	}

	return count, breadcrumbs
}

func (agent *AgentAPI) GetBuffer(buffer_id int) []byte {
	buffer_size := int(agent.c_api.mgr.meta.buffer_size)
	start := buffer_id * buffer_size
	end := start + buffer_size
	var data []byte
	data = (*[1<<30]byte)(unsafe.Pointer(agent.c_api.mgr.pool))[start:end]
	return data
}

func (api *GoAgentAPI) GetBuffer(buffer_id int) []byte {
	return api.agent.GetBuffer(buffer_id)
}
