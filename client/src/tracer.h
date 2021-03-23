#ifndef _TRACER_H_
#define _TRACER_H_

#include <stdlib.h>
#include <inttypes.h>
#include <stdbool.h>

#include "queue.h"

// Trace Data Structures

typedef struct Medadata {
	uint64_t request_id;
	uint64_t span_id;
	uint64_t parent_span_id;
} Metadata;

typedef const char* AgentAddress;

typedef struct Header {
	Metadata trace_md;
	AgentAddress breadcrumbs[8];
	int bredcrumb_count;
} Header;

typedef struct Buffer {
	int buffer_id;
	void* ptr;
	size_t offset;
} Buffer;

typedef struct Pool {
	void* ptr;
	size_t pool_size;
	const size_t buffer_length;
} Pool;

typedef struct SendQueue {
	queue_handle_t queue;
} SendQueue;

typedef struct RecvQueue {
	queue_handle_t queue;
} RecvQueue;


// Global Structures

extern Pool pool;
extern SendQueue complete;
extern RecvQueue available;
extern SendQueue triggers;

// Thread Local Variables
extern __thread bool active;
extern __thread Header header;
extern __thread Buffer buffer;


// Tracer APIs

void flush(Metadata metadata);

void trace_init();

void trace_begin(Metadata metadata);

void trace_end();

void tracepoint();

void trace_add_breadcrumb(AgentAddress breadcrumb);

Metadata trace_get_metadata();

uint64_t trace_get_request_id();

uint64_t trace_get_span_id();

void trace_set_span_id();

void trace_set_parent_span_id();

// Rate Limiters

#endif
