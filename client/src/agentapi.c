#include "agentapi.h"

#include <assert.h>

HindsightAgentAPI* hindsight_agentapi_init(const char* servicename) {
    HindsightAgentAPI* api = malloc(sizeof(HindsightAgentAPI));
    api->mgr = bufmanager_init_existing(servicename);
    api->triggers = triggers_init_existing(servicename);
    api->breadcrumbs = breadcrumbs_init_existing(servicename);
    return api;
}

// Return a batch of `buffers->count` (<BATCHSIZE) buffers to the available queue.
// Blocks until all buffers can be returned to the queue.
void hindsight_agentapi_put_available_blocking(HindsightAgentAPI* api, AvailableBuffers* buffers) {
    assert(buffers->count <= BATCHSIZE);
    assert(buffers->count > 0);

    queue_put_blocking_multi(&api->mgr.available, (char*) buffers->bufs, buffers->count);
}

void hindsight_agentapi_get_available_nonblocking(HindsightAgentAPI* api, AvailableBuffers* buffers) {
    buffers->count = queue_get_nonblocking_multi(&api->mgr.available, (char*) buffers->bufs, BATCHSIZE);
}

// Retrieves a batch of up to BATCHSIZE buffers from the complete queue.
// Returns between 0 and BATCHSIZE buffers
void hindsight_agentapi_get_complete_nonblocking(HindsightAgentAPI* api, CompleteBuffers* buffers) {
    buffers->count = queue_get_nonblocking_multi(&api->mgr.complete, (char*) buffers->bufs, BATCHSIZE);
}

void hindsight_agentapi_get_triggers_nonblocking(HindsightAgentAPI* api, TriggerBatch* triggers) {
    triggers->count = queue_get_nonblocking_multi(&api->triggers, (char*) triggers->triggers, BATCHSIZE);
}

void hindsight_agentapi_get_breadcrumbs_nonblocking(HindsightAgentAPI* api, BreadcrumbBatch* breadcrumbs) {
    breadcrumbs->count = queue_get_nonblocking_multi(&api->breadcrumbs, (char*) breadcrumbs->breadcrumbs, BATCHSIZE);
}