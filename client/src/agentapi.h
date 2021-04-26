#ifndef _HINDSIGHT_CLIENT_AGENTAPI_H_
#define _HINDSIGHT_CLIENT_AGENTAPI_H_

#include <stddef.h>
#include <stdbool.h>

#include "buffer.h"
#include "breadcrumb.h"
#include "trigger.h"
#include "hindsight.h"

#define BATCHSIZE 100

/*
This is a C implementation of the agent-side interface
to Hindsight's shared-memory bits.  It's barebones --
all it provides is an API to read data and queues.
*/
typedef struct HindsightAgentAPI {
	BufManager mgr;
	Triggers triggers;
	Breadcrumbs breadcrumbs;
} HindsightAgentAPI;

// Initialize the agent API for a service.
// This call will block if the specified service doesn't exist,
// waiting until it has started.
// Configurations will be read from shared memory.
HindsightAgentAPI* hindsight_agentapi_init(const char* servicename);

typedef struct AvailableBuffers {
	size_t count; // up to BATCHSIZE allowed at a time
	AvailableBuffer bufs[BATCHSIZE]; // hard-coded to BATCHSIZE for ease of use with go
} AvailableBuffers;

typedef struct CompleteBuffers {
	size_t count; // will return up to BATCHSIZE at a time
	CompleteBuffer bufs[BATCHSIZE]; // hard-coded to BATCHSIZE for ease of use with go
} CompleteBuffers;

// Return a batch of `buffers->count` (<BATCHSIZE) buffers to the available queue.
// Blocks until all buffers can be returned to the queue.
void hindsight_agentapi_put_available_blocking(HindsightAgentAPI* api, AvailableBuffers* buffers);

// Retrieves a batch of up to BATCHSIZE buffers from the complete queue.
// Returns between 0 and BATCHSIZE buffers
void hindsight_agentapi_get_complete_nonblocking(HindsightAgentAPI* api, CompleteBuffers* buffers);




#endif // _HINDSIGHT_CLIENT_AGENTAPI_H_