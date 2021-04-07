#ifndef _TRACER_H_
#define _TRACER_H_

#include <stdlib.h>
#include <inttypes.h>
#include <stdbool.h>

#include "queue.h"
#include "memory.h"

// Trace Data Structures

/*
	Buffer data structure:
	variable 		type 		size
	req_id  		uint64_t 	8
	time 			uint64_t 	8
	span_id 		uint64_t 	8
	parent_span_id 	uint64_t 	8
	bread_count 	int 		4
	breadcrumbs 	int*8 		4*8
	data 			int*x 		4*x

	Buffer entry data structure:
	id 				int 		4
	payload(n) 		int*n 		4*n
*/

typedef struct Buffer {
	int buffer_id;
	int offset;
	int* ptr;
} Buffer;

// typedef struct Pool {
// 	int size;
// 	int buffer_length;
// 	int* ptr;
// } Pool;

typedef int* Pool;

/*
	char*, 32*100
*/
typedef char* Dictionary;

typedef struct Medadata {
	uint64_t request_id;
	uint64_t timestamp;
	uint64_t span_id;
	uint64_t parent_span_id;
} Metadata;

typedef const char* AgentAddress;

typedef struct Header {
	Metadata* trace_md;
	int* breadcrumbs;
	int breadcrumb_count;
} Header;

typedef struct SendQueue {
	Queue queue;
} SendQueue;

typedef struct RecvQueue {
	Queue queue;
} RecvQueue;


// Global Structures

extern Pool pool;
extern int pool_size;
extern int pool_buffer_length;
extern SendQueue complete;
extern RecvQueue available;
extern SendQueue triggers;

extern Dictionary dictionary;
extern int dict_count;

// Thread Local Variables
extern __thread bool active;
extern __thread Header *header;
extern __thread Buffer *buffer;


// Tracer APIs

void flush();

time_t get_time();

void trace_init(int cap);

void trace_begin(uint64_t request_id, uint64_t span_id, uint64_t parent_span_id);

void trace_end();

void tracepoint(int id, int payload);

void trace_add_breadcrumb(AgentAddress breadcrumb);

Metadata* trace_get_metadata();

uint64_t trace_get_request_id();

uint64_t trace_get_span_id();

void trace_set_span_id(uint64_t span_id);

void trace_set_parent_span_id(uint64_t parent_span_id);

// Rate Limiters

#endif
