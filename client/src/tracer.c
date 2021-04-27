#define _GNU_SOURCE
#include <time.h>
#include <string.h>
#include <sys/mman.h>
#include <fcntl.h>
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <sched.h>
#include <string.h>


#include "tracer.h"
#include "hindsight.h"

// Legacy span ID and parent span ID; not needed by hindsight
__thread uint64_t span_id;
__thread uint64_t parent_span_id;

// Queue Handler APIs

void trigger(uint64_t request_id_){
    // Hindsight supports a trigger ID too; not used here
    hindsight_trigger_manual(request_id_, 0);
}

void trace_init(const char* service_name){
    hindsight_init(service_name);
}

void trace_begin(uint64_t request_id_, uint64_t span_id_, uint64_t parent_span_id_){
    // Hindsight doesn't need span ID or parent span ID, but leaving them here for posterity
    hindsight_begin(request_id_);
    span_id = span_id_;
    parent_span_id = parent_span_id_;
}

void trace_end(){
    hindsight_end();
}

void tracepoint(int id, int payload){
    // Only implementing this API for backwards compatibility
    int buf[payload];
    for (int i = 0; i < payload; i++) {
        buf[i] = id;
    }
}

void trace_add_breadcrumb(AgentAddress breadcrumb){
    hindsight_breadcrumb((const char*) breadcrumb);
}

uint64_t trace_get_request_id(){
    return hindsight_get_traceid();
}

uint64_t trace_get_span_id(){
    return span_id;
}

void trace_set_span_id(uint64_t span_id_){
    span_id = span_id_;
}

void trace_set_parent_span_id(uint64_t parent_span_id_){
    parent_span_id = parent_span_id_;
}

char* serialize() {
    return hindsight_get_local_address();
}

void deserialize(char* baggage) {
    hindsight_breadcrumb(baggage);
}