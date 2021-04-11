#define _GNU_SOURCE
#include <time.h>
#include <string.h>
#include <sys/mman.h>
#include <fcntl.h>
#include <assert.h>
#include <stdio.h>
#include <unistd.h>
#include <sys/types.h>


#include "tracer.h"
#include "queue.h"

// Global Structures
Pool pool;
int pool_size;
int pool_buffer_length;
SendQueue* complete;
RecvQueue* available;
SendQueue* triggers;

Dictionary dictionary;
int dict_count;

// Thread Local Variables
__thread bool active;
__thread bool first_buf;
__thread Header *header;
__thread Buffer *buffer;

// Queue Handler APIs
void trigger(uint64_t trigger_id){
	return;
}

int acquire(){
	return queue_get(available->queue);
}

void release(int buffer_id){
	queue_put(complete->queue, buffer_id);
	return;
}

void* mem_init(const char* fname, size_t fsize) {
	void* shm;
	
	int fd = open(fname, O_RDWR | O_CREAT, 0666);
	assert(fd >= 0);

	// int isExist = isFileExist(fname);

	int i = ftruncate(fd, fsize);
	assert(i == 0);

	shm = mmap(NULL, fsize, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
	assert(shm != MAP_FAILED);	
	close(fd);

	memset(shm, 0, fsize);

	return shm;
}

bool isFileExist(const char* fname) {
	return (access(fname, F_OK) != -1);
}


// Tracer APIs

void flush() {
	if (active == true) {
		int offset = buffer->buffer_id * pool_buffer_length;
		pool[offset+1] = (int)(header->trace_md->request_id >> 32);
		pool[offset] = (int)(header->trace_md->request_id & 0xffffffff);
		if (first_buf) {
			pool[offset+3] = (int)(header->trace_md->timestamp >> 32);
			pool[offset+2] = (int)(header->trace_md->timestamp & 0xffffffff);
			pool[offset+5] = (int)(header->trace_md->span_id >> 32);
			pool[offset+4] = (int)(header->trace_md->span_id & 0xffffffff);
			pool[offset+7] = (int)(header->trace_md->parent_span_id >> 32);
			pool[offset+6] = (int)(header->trace_md->parent_span_id & 0xffffffff);
		}
		pool[offset+8] = header->breadcrumb_count;
		for (int i=0; i<8; i++) {
			pool[offset+9+i] = header->breadcrumbs[i];
		}
		for (int i=17; i<pool_buffer_length; i++) {
			pool[offset+i] = buffer->ptr[i];
		}
	}
	release(buffer->buffer_id);
	active = false;
	return;
}

time_t get_time() {
	struct timespec ts;
	timespec_get(&ts, TIME_UTC);
	return ts.tv_sec * 1000000000 + ts.tv_nsec;
}

void trace_init(int cap){
	pool = (int*)mem_init("/dev/shm/pool", cap*50*sizeof(int));
	pool_size = cap;
	pool_buffer_length = 50;
	// pool[pool_size] = pool_size; // pool->size
	// pool[pool_size+1] = pool_buffer_length; // pool->buffer_length, header takes 17, must more than it

	complete = malloc(sizeof(SendQueue));
	available = malloc(sizeof(RecvQueue));
	triggers = malloc(sizeof(SendQueue));

	available->queue = (Queue)queue_init("/dev/shm/available_queue", cap);
	complete->queue = (Queue)queue_init("/dev/shm/complete_queue", cap);
	triggers->queue = (Queue)queue_init("/dev/shm/triggers_queue", cap);

	// avail queue initialization should be in agent
	// for (int i=0; i<pool_size; i++) queue_put(available->queue, i);

	dictionary = (char*)mem_init("/dev/shm/dict", 3200);
	dict_count = 0;

	active = false;	
	first_buf = true;

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
	first_buf = true;
	if (buf == -1) {
		active = false;
		#if(DEBUG)
			printf("no buffer acquired\n");
		#endif
		return;
	}
	
	active = true;
	
	buffer->buffer_id = buf;
	buffer->offset = 17;
	#if(DEBUG)
		printf("[trace_begin] buffer %d\n", buf);
	#endif

	header->trace_md->request_id = request_id;
	header->trace_md->timestamp = get_time();
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
	#if(DEBUG)
		printf("[trace_end]\n");
	#endif
	return;
}

void tracepoint(int id, int payload){
	// write to buffer->ptr, allow partial data of payloads across buffers
	for (int i=0; i<payload + 1; i++) {
		if (buffer->offset >= pool_buffer_length) {
			#if(DEBUG)
				printf("[tracepoint]acquire new buffer when offset %d >= buffer length %d\n", buffer->offset, pool_buffer_length);
			#endif
			flush();
			int buf = acquire();
			first_buf = false;
			
			active = true;
			buffer->buffer_id = buf;
			buffer->offset = 17;
			for (int j=17; j<pool_buffer_length; j++) {
				buffer->ptr[j] = 0;
			}
		}

		if(i==0){
			buffer->ptr[buffer->offset] = id;
			buffer->offset++;
		}
		else {
			buffer->ptr[buffer->offset] = i;
			buffer->offset++;
		}

		#if(DEBUG)
			printf("[tracepoint]writing payload %d in buffer %d offset %d\n", i, buffer->buffer_id, buffer->offset);
		#endif
	}

	return;
}

void trace_add_breadcrumb(AgentAddress breadcrumb){
	// write to dictionary
	int offset = dict_count * 32;
	memcpy(dictionary+offset, breadcrumb, strlen(breadcrumb));
	#if(DEBUG)
		printf("[trace_add_breadcrumb]add at %d\n", dict_count);
	#endif

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