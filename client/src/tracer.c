#define _GNU_SOURCE
#include <time.h>
#include <string.h>

#include "tracer.h"

// Global Structures

Pool pool;
int pool_size;
int pool_buffer_length;
SendQueue complete;
RecvQueue available;
SendQueue triggers;

Dictionary dictionary;
int dict_count;

// Thread Local Variables
__thread bool active;
__thread Header *header;
__thread Buffer *buffer;


// Tracer APIs

void flush() {
	if (active == true) {
		int offset = buffer->buffer_id * pool_size;
		pool[offset] = (int)(header->trace_md->request_id >> 32);
		pool[offset+1] = (int)(header->trace_md->request_id & 0xffffffff);
		pool[offset+2] = 0; // time
		pool[offset+4] = (int)(header->trace_md->span_id >> 32);
		pool[offset+5] = (int)(header->trace_md->span_id & 0xffffffff);
		pool[offset+6] = (int)(header->trace_md->parent_span_id >> 32);
		pool[offset+7] = (int)(header->trace_md->parent_span_id & 0xffffffff);
		pool[offset+8] = header->breadcrumb_count;
		for (int i=0; i<8; i++) {
			pool[offset+9+i] = header->breadcrumbs[i];
		}
		for (int i=0; i<buffer->offset; i++) {
			pool[offset+17+i] = buffer->ptr[i];
		}
	}
	active = false;
	return;
}

time_t get_time() {
	struct timespec ts;
	timespec_get(&ts, TIME_UTC);
	return ts.tv_sec * 1000000000 + ts.tv_nsec;
}

void trace_init(int cap){
	pool = (int*)mem_init("/dev/shm/pool_test", (cap+2)*sizeof(int));
	pool_size = cap;
	pool_buffer_length = 50;
	pool[pool_size] = cap; // pool->size
	pool[pool_size+1] = 50; // pool->buffer_length, header takes 17, must more than it


	// TODO: queue initializations (open|create)
	// TODO: Dictionary
	dictionary = (char*)mem_init("/dev/shm/dict", 3200);
	dict_count = 0;

	active = false;	

	buffer = malloc(sizeof(Buffer));
	buffer->buffer_id = 0;
	buffer->offset = 0;
	buffer->ptr = malloc(sizeof(int)*50);

	header = malloc(sizeof(Header));
	header->trace_md = malloc(sizeof(Metadata));
	header->breadcrumbs = malloc(sizeof(int)*8);
	header->breadcrumb_count = 0;

	return;
}

void trace_begin(uint64_t request_id, uint64_t span_id, uint64_t parent_span_id){
	if (active) {
		flush();
	}
	int buf = acquire();
	if (buf == -1) {
		active = false;
		printf("no buffer acquired\n");
		return;
	}
	
	active = true;
	
	buffer->buffer_id = buf;
	buffer->offset = 0;

	header->trace_md->request_id = request_id;
	header->trace_md->timestamp = 0;
	header->trace_md->span_id = span_id;
	header->trace_md->parent_span_id = parent_span_id;

	for (int i=0; i<8; i++) {
		header->breadcrumbs[i] = 0;
	}
	header->breadcrumb_count = 0;

	return;
}

void trace_end(){
	if (active == true) {
		flush();
		active = false;
	}
	return;
}

void tracepoint(int id, int payload){
	// write to buffer->ptr, allow partial data of payloads across buffers
	for (int i=0; i<payload + 1; i++) {
		if (buffer->offset > pool[pool_size+1]) {
			flush();
			int buf = acquire();
			buffer->buffer_id = buf;
			buffer->offset = 0;
		}

		if(i==0){
			buffer->ptr[buffer->offset] = id;
			buffer->offset++;
		}
		else {
			buffer->ptr[buffer->offset] = i;
			buffer->offset++;	
		}
	}

	return;
}

void trace_add_breadcrumb(AgentAddress breadcrumb){
	// write to dictionary
	int offset = dict_count * 32;
	memcpy(dictionary+offset, breadcrumb, strlen(breadcrumb));
	if (header->breadcrumb_count < 8)
		header->breadcrumbs[header->breadcrumb_count] = dict_count; 
	header->breadcrumb_count++;
	dict_count++;

	return;
}

Metadata* trace_get_metadata(){
	return header->trace_md;
}

uint64_t trace_get_request_id(){
	return header->trace_md->request_id;
}

uint64_t trace_get_span_id(){
	return header->trace_md->span_id;
}

void trace_set_span_id(uint64_t span_id){
	header->trace_md->span_id = span_id;
	return;
}

void trace_set_parent_span_id(uint64_t parent_span_id){
	header->trace_md->parent_span_id = parent_span_id;
	return;
}