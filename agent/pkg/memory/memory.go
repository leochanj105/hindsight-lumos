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
    agent.drainAllBuffers()
    agent.releaseAllBuffers()
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

func (api *GoAgentAPI) completeLoop(ctx context.Context) {
    max_backoff := 100000
    backoff := int(10)
	for {
		select {
		case <- ctx.Done():
			return
		default:
			completed := api.agent.GetCompleteBatches()
			if len(completed) > 0 {
				api.Complete <- completed
				backoff = int(10)
			} else {
				duration := time.Duration(backoff) * time.Microsecond
				time.Sleep(duration)
				backoff *= 2
		        if (backoff > max_backoff) {
		            backoff = max_backoff
		        }
			}
		}
	}
}

func (api *GoAgentAPI) triggerLoop(ctx context.Context) {
    max_backoff := 100000
    backoff := int(10)
	for {
		select {
		case <- ctx.Done():
			return
		default:
			triggers := api.agent.GetTriggers()
			if len(triggers) > 0 {
				api.Triggers <- triggers
				backoff = int(10)
			} else {
				time.Sleep(time.Duration(backoff) * time.Nanosecond)
				backoff *= 2
	            if (backoff > max_backoff) {
	                backoff = max_backoff
	            }
			}
		}
	}
}

func (api *GoAgentAPI) breadcrumbsLoop(ctx context.Context) {
    max_backoff := 100000
    backoff := int(10)
	for {
		select {
		case <- ctx.Done():
			return
		default:
			breadcrumbs := api.agent.GetBreadcrumbBatches()
			if len(breadcrumbs) > 0 {
				api.Breadcrumbs <- breadcrumbs
				backoff = int(10)
			} else {
				time.Sleep(time.Duration(backoff) * time.Nanosecond)
				backoff *= 2
	            if (backoff > max_backoff) {
	                backoff = max_backoff
	            }
			}
		}
	}
}


func (agent *AgentAPI) drainAllBuffers() {
	// Drain any available buffers
    davailable := 0
    for {
        var ab C.AvailableBuffers
        C.hindsight_agentapi_get_available_nonblocking(agent.c_api, &ab)

        if (ab.count == 0) {
            break
        }

        davailable += int(ab.count)
    }

    // Drain all complete buffers
    dcomplete := 0
    for {
        var cb C.CompleteBuffers
        C.hindsight_agentapi_get_complete_nonblocking(agent.c_api, &cb)

        if (cb.count == 0) {
            break
        }

        dcomplete += int(cb.count)
    }

    fmt.Println("Resetting buffers: drained", davailable, "available and", dcomplete, "complete")
}

// Makes all buffers available in the shm available queue
// TODO: initial buffers should be made available by the client rather than agent probably
func (agent *AgentAPI) releaseAllBuffers() {
    fmt.Println("Initialize buffers: making", agent.c_api.mgr.meta.capacity, "buffers available...")

    next_buffer_id := 0
    remaining := agent.c_api.mgr.meta.capacity
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

        C.hindsight_agentapi_put_available_blocking(agent.c_api, &av);
    }

    fmt.Println("Initialize buffers: done")
    fmt.Println("Queue states:")
    fmt.Print("  Available ")
    C.queue_print(&agent.c_api.mgr.available)
    fmt.Print("  Complete ")
    C.queue_print(&agent.c_api.mgr.complete)
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
func (agent *AgentAPI) GetCompleteBatches() CompleteBatch {
	var cb C.CompleteBuffers
	C.hindsight_agentapi_get_complete_nonblocking(agent.c_api, &cb)

	count := int(cb.count)
	buffers := make(CompleteBatch, count)
	for i := 0; i < count; i++ {
		trace_id := uint64(cb.bufs[i].trace_id)
		buffer_id := int(cb.bufs[i].buffer_id)
		buffers[trace_id] = append(buffers[trace_id], buffer_id)
	}

	return buffers
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
func (agent *AgentAPI) GetBreadcrumbBatches() BreadcrumbBatch {
	var bb C.BreadcrumbBatch
	C.hindsight_agentapi_get_breadcrumbs_nonblocking(agent.c_api, &bb)

	count := int(bb.count)
	breadcrumbs := make(BreadcrumbBatch, count)
	for i := 0; i < count; i++ {
		trace_id := uint64(bb.breadcrumbs[i].trace_id)
		addr := C.GoString(bb.breadcrumb_addrs[i])
		breadcrumbs[trace_id] = append(breadcrumbs[trace_id], addr)
	}

	return breadcrumbs	
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
