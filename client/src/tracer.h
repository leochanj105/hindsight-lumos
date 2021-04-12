#ifndef _TRACER_H_
#define _TRACER_H_

#include <stdlib.h>
#include <inttypes.h>
#include <stdbool.h>
#include <time.h>
#include <x86intrin.h>

#ifndef DEBUG
#define DEBUG 0
#endif

#include "queue.h"

#define TRACEBEGIN(x,y,z) trace_begin(x,y,z)
#define TRACEPOINT(x,y) tracepoint(x,y)
#define TRACEEND() trace_end()

// Trace Data Structures

// typedef struct Pool {
// 	int size;
// 	int buffer_length;
// 	int* ptr;
// } Pool;

typedef int* Pool;

/*
	Each buffer data entry structure in pool
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

typedef struct Medadata {
	uint64_t request_id;
	uint64_t timestamp;
	uint64_t span_id;
	uint64_t parent_span_id;
} Metadata;

typedef struct Header {
	Metadata* trace_md;
	int* breadcrumbs;
	int breadcrumb_count;
} Header;

typedef struct Buffer {
	int buffer_id;
	int offset;
	int* ptr;
} Buffer;

/*
	char*, 32*100
*/
typedef char* Dictionary;

typedef const char* AgentAddress;

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
extern SendQueue* complete;
extern RecvQueue* available;
extern SendQueue* triggers;

extern Dictionary dictionary;
extern int dict_count;

// Thread Local Variables
extern __thread bool active;
extern __thread bool first_buf;
// extern __thread Header* header;
// extern __thread Buffer* buffer;

extern __thread int buffer_id;
extern __thread int buffer_offset;
extern __thread int buffer_ptr[50];

extern __thread uint64_t request_id;
extern __thread uint64_t timestamp;
extern __thread uint64_t span_id;
extern __thread uint64_t parent_span_id;
extern __thread int breadcrumbs[8];
extern __thread int breadcrumb_count;


// Queue handler APIs

void trigger(uint64_t trigger_id);

int acquire();

void release(int buffer_id);

void* mem_init(const char* fname, size_t fsize);

bool isFileExist(const char* fname);

// Tracer APIs

void flush();

time_t get_time();

void trace_init(int cap);

void trace_begin(uint64_t request_id_, uint64_t span_id_, uint64_t parent_span_id_);

void trace_end();

void tracepoint(int id, int payload);

void trace_add_breadcrumb(AgentAddress breadcrumb);

// Metadata* trace_get_metadata();

uint64_t trace_get_request_id();

uint64_t trace_get_span_id();

void trace_set_span_id(uint64_t span_id_);

void trace_set_parent_span_id(uint64_t parent_span_id_);

// Rate Limiters

void trace_test(uint64_t temp);

#endif
