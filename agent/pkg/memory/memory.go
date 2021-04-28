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
)

// BATCHSIZE is #defined in agentapi.h
//   The value here must be the same as the value in agentapi.h
const BATCHSIZE = 100

type AgentAPI struct {
	c_api *C.HindsightAgentAPI
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
func (agent *AgentAPI) GetTriggers() []*Trigger {
	var tb C.TriggerBatch
	C.hindsight_agentapi_get_triggers_nonblocking(agent.c_api, &tb)

	var triggers []*Trigger
	count := int(tb.count)
	for i := 0; i < count; i++ {
		var trigger Trigger
		trigger.Request_id = uint64(tb.triggers[i].trace_id)
		trigger.Trigger_id = int(tb.triggers[i].trigger_id)
		triggers = append(triggers, &trigger)
	}

	return triggers
}


/* Retrieves up to BATCHSIZE breadcrumbs from the breadcrumbs queue.

BATCHSIZE is hard-coded in agentapi.h

This is a non-blocking call; may return 0 breadcrumbs
*/
func (agent *AgentAPI) GetBreadcrumbs() []*Breadcrumb {
	var bb C.BreadcrumbBatch
	C.hindsight_agentapi_get_breadcrumbs_nonblocking(agent.c_api, &bb)

	var breadcrumbs []*Breadcrumb
	count := int(bb.count)
	for i := 0; i < count; i++ {
		var breadcrumb Breadcrumb
		breadcrumb.Request_id = uint64(bb.breadcrumbs[i].trace_id)
		breadcrumb.Address = C.GoString(bb.breadcrumb_addrs[i])
		breadcrumbs = append(breadcrumbs, &breadcrumb)
	}

	return breadcrumbs	
}
