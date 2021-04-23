#ifndef _HINDSIGHT_CLIENT_AGENTAPI_H_
#define _HINDSIGHT_CLIENT_AGENTAPI_H_

#include <stddef.h>
#include <stdbool.h>

#include "buffer.h"
#include "breadcrumb.h"
#include "trigger.h"
#include "hindsight.h"

/*
This is a C implementation of the agent-side interface
to Hindsight's shared-memory bits.  It's barebones --
all it provides is an API to read data and queues.
*/
typedef struct HindsightAgentAPI {
	HindsightConfig config;
	
	Breadcrumbs breadcrumbs;
	Triggers triggers;
} HindsightAgentAPI;

// Initialize the agent API 
HindsightAgentAPI hindsight_agentapi_init(const char* servicename);




#endif // _HINDSIGHT_CLIENT_AGENTAPI_H_