#ifndef _TRACER_H_
#define _TRACER_H_

#include <stdlib.h>
#include <inttypes.h>
#include <stdbool.h>
#include <time.h>
#include <x86intrin.h>

#define TRACEINIT(x) 		trace_init(x)

#define TRACEBEGIN(x) 	trace_begin(x,0,0)
#define TRACEEND()  		trace_end()
#define TRACEPOINT(x,y) 	tracepoint(x,y)
#define TRIGGER(x) 			trigger(x)
#define TRACEBREADCRUMB(x) 	trace_add_breadcrumb(x)

#define SERIALIZE() 		serialize()
#define DESERIALIZE() 		deserialize()


typedef const char* AgentAddress;

void trigger(uint64_t request_id_);

void trace_init(const char* service_name);

void trace_begin(uint64_t request_id_, uint64_t span_id_, uint64_t parent_span_id_);

void trace_end();

void tracepoint(int id, int payload);

void trace_add_breadcrumb(AgentAddress breadcrumb);

uint64_t trace_get_request_id();

uint64_t trace_get_span_id();

void trace_set_span_id(uint64_t span_id_);

void trace_set_parent_span_id(uint64_t parent_span_id_);

char* serialize();

void deserialize(char* baggage);

#endif
