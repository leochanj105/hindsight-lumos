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


#include "tracer.h"
#include "queue.h"

// Global Structures
Pool pool;
int pool_cap;
int pool_buffer_length;
char* service_addr;
char* service_port;
SendQueue* complete;
RecvQueue* available;
SendQueue* triggers;
char* trigger_lock;

Dictionary dictionary;
int dict_count;

// Thread Local Variables
__thread bool active;
__thread bool first_buf;
// __thread Header* header;
// __thread Buffer* buffer;

__thread int buffer_id;
__thread int buffer_offset;
__thread int buffer_ptr[50];

__thread uint64_t request_id;
__thread uint64_t timestamp;
__thread uint64_t span_id;
__thread uint64_t parent_span_id;
__thread int breadcrumbs[8];
__thread int breadcrumb_count;


// Queue Handler APIs
void trigger(uint64_t request_id_){
	// trigger needs to write an int64 to queue as two entries
	Lock(trigger_lock);
	queue_put(triggers->queue, (int)(request_id_ >> 32));
	queue_put(triggers->queue, (int)(request_id_ & 0xffffffff));
	Unlock(trigger_lock);
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
		#if(DEBUG)
			printf("[tracer] flush to buffer %d\n", buffer_id);
		#endif
		int offset = buffer_id * pool_buffer_length;
		pool[offset+1] = (int)(request_id >> 32);
		pool[offset] = (int)(request_id & 0xffffffff);
		if (first_buf) {
			pool[offset+3] = (int)(timestamp >> 32);
			pool[offset+2] = (int)(timestamp & 0xffffffff);
			pool[offset+5] = (int)(span_id >> 32);
			pool[offset+4] = (int)(span_id & 0xffffffff);
			pool[offset+7] = (int)(parent_span_id >> 32);
			pool[offset+6] = (int)(parent_span_id & 0xffffffff);
		}
		// pool[offset+8] = breadcrumb_count;
		// for (int i=0; i<8; i++) {
		// 	pool[offset+9+i] = breadcrumbs[i];
		// }
		// pool[offset+17] = buffer_offset;
		// for (int i=18; i<pool_buffer_length; i++) {
		// 	pool[offset+i] = buffer_ptr[i];
		// }
	}
	release(buffer_id);
	active = false;
	return;
}

time_t get_time() {
	struct timespec ts;
	timespec_get(&ts, TIME_UTC);
	return ts.tv_sec * 1000000000 + ts.tv_nsec;	
}

void load_config(const char* fname) {
	// load default value first
	pool_cap = 1;
	pool_buffer_length = 1;
	service_addr = malloc(32*sizeof(char));
	service_port = malloc(32*sizeof(char));
	memset(service_addr, 0, 32*sizeof(char));
	memset(service_port, 0, 32*sizeof(char));
	
	FILE* config_file;
	config_file = fopen(fname,"r");
	if (config_file == NULL) {
		config_file = fopen("/etc/hindsight_conf/default.conf","r");
	}

	char* line = NULL;
	// if (infile == NULL) {
	// 	infile = fopen("/etc/hindsight.conf", "r");
	// }

	ssize_t read;
	size_t len = 0;

	while((read = getline(&line, &len, config_file)) != -1) {
		char* temp = strchr(line, '\n');
		int index = (int)(temp - line);

		char* new_line = malloc(sizeof(char)*20);
		if (index == strlen(line)-1) {
			strncpy(new_line, line, index);
		} else {
			strncpy(new_line, line, strlen(line));
		}

		char* var = malloc(sizeof(char)*20);
		char* value = malloc(sizeof(char)*20);
		sscanf(new_line, "%s %s", var, value);

		if (!strcmp(var, "cap")) {
			pool_cap = atoi(value);
		}

		if (!strcmp(var, "buf_length")) {
			pool_buffer_length = atoi(value);
		}

		if (!strcmp(var, "addr")) {
			service_addr = value;
		}

		if (!strcmp(var, "port")) {
			service_port = value;
		}		
	}
	fclose(config_file);

	if (line) free(line);

	printf("config file load cap=%d buffer_length=%d service_addr=%s, service_port=%s\n", pool_cap, pool_buffer_length, service_addr, service_port);

	return;

}

char* get_fname(char* dst1, char* dst2) {
	char* name = malloc(sizeof(char)*64);
	memset(name, 0, sizeof(char)*64);
	strcpy(name, dst1);
	strcat(name, dst2);
	return name;
}

void trace_init(const char* service_name){
	char config_fname[64];
	strcpy(config_fname, "/etc/hindsight_conf/");
	strcat(config_fname, service_name);
	strcat(config_fname, ".conf");

	load_config(config_fname);

	pool = (int*)mem_init(get_fname("/dev/shm/pool_", service_name), pool_cap*pool_buffer_length*sizeof(int));
	
	// pool_cap = cap;
	// pool_buffer_length = 50;
	

	complete = malloc(sizeof(SendQueue));
	available = malloc(sizeof(RecvQueue));
	triggers = malloc(sizeof(SendQueue));

	available->queue = (Queue)queue_init(get_fname("/dev/shm/available_queue_", service_name), pool_cap);
	complete->queue = (Queue)queue_init(get_fname("/dev/shm/complete_queue_", service_name), pool_cap);
	triggers->queue = (Queue)queue_init(get_fname("/dev/shm/triggers_queue_", service_name), pool_cap);
	trigger_lock = (char*)malloc(sizeof(char));
	memset(trigger_lock, '0', sizeof(char));

	dictionary = (char*)mem_init(get_fname("/dev/shm/dict_", service_name), 3200);
	dict_count = 0;

	active = false;	
	first_buf = true;

	// buffer = (Buffer*)malloc(sizeof(Buffer));
	// buffer->buffer_id = 0;
	// buffer->offset = 0;
	// buffer->ptr = (int*)malloc(sizeof(int)*50);

	// header = (Header*)malloc(sizeof(Header));
	// header->trace_md = (Metadata*)malloc(sizeof(Metadata));
	// header->breadcrumbs = (int*)malloc(sizeof(int)*8);
	// header->breadcrumb_count = 0;

	buffer_id = 0;
	buffer_offset = 0;
	// buffer_ptr = (int*)malloc(sizeof(int)*50);

	request_id = (uint64_t)0;
	timestamp = (uint64_t)0;
	span_id = (uint64_t)0;
	parent_span_id = (uint64_t)0;
	// breadcrumbs = (int*)malloc(sizeof(int)*8);
	breadcrumb_count = 0;

	return;
}

void trace_begin(uint64_t request_id_, uint64_t span_id_, uint64_t parent_span_id_){
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
	
	#if(DEBUG)
		printf("[trace_begin] buffer %d\n", buf);
	#endif

	buffer_id = buf;
	buffer_offset = 18;

	request_id = request_id_;
	timestamp = get_time();
	span_id = span_id_;
	parent_span_id = parent_span_id_;

	// for (int i=0; i<8; i++) {
	// 	breadcrumbs[i] = 0;
	// }
	breadcrumb_count = 0;

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
		if (buffer_offset >= pool_buffer_length) {
			#if(DEBUG)
				printf("[tracepoint]acquire new buffer when offset %d >= buffer length %d\n", buffer_offset, pool_buffer_length);
			#endif
			flush();
			int buf = acquire();
			first_buf = false;
			
			active = true;
			buffer_id = buf;
			buffer_offset = 18;
			for (int j=18; j<pool_buffer_length; j++) {
				buffer_ptr[j] = 0;
			}
		}

		if(i==0){
			buffer_ptr[buffer_offset] = id;
			buffer_offset++;
		}
		else {
			buffer_ptr[buffer_offset] = i;
			buffer_offset++;
		}

		#if(DEBUG)
			printf("[tracepoint]writing payload %d in buffer %d offset %d\n", i, buffer_id, buffer_offset);
		#endif
	}

	return;
}

void trace_add_breadcrumb(AgentAddress breadcrumb){
	// check if coming address is already known
	for (int i=0; i<dict_count; i++) {
		int offset = i * 32;
		if (strncmp(dictionary+offset, breadcrumb, strlen(breadcrumb)) == 0) {
			if (breadcrumb_count < 8)
				breadcrumbs[breadcrumb_count] = i; 
			breadcrumb_count++;
			return;
		}
	}

	// write to dictionary
	int offset = dict_count * 32;
	strncpy(dictionary+offset, breadcrumb, strlen(breadcrumb));
	#if(DEBUG)
		printf("[trace_add_breadcrumb]add at %d\n", dict_count);
	#endif

	if (breadcrumb_count < 8)
		breadcrumbs[breadcrumb_count] = dict_count; 
	breadcrumb_count++;
	dict_count++;

	return;
}

// Metadata* trace_get_metadata(){
// 	return header->trace_md;
// }

uint64_t trace_get_request_id(){
	return request_id;
}

uint64_t trace_get_span_id(){
	return span_id;
}

void trace_set_span_id(uint64_t span_id_){
	span_id = span_id_;
	return;
}

void trace_set_parent_span_id(uint64_t parent_span_id_){
	parent_span_id = parent_span_id_;
	return;
}

void trace_test(uint64_t temp) {
	buffer_id = (int)temp;
	buffer_offset = (int)temp;

	request_id = temp;
	timestamp = temp;
	span_id = temp;
	parent_span_id = temp;

	for (int i=0; i<8; i++) {
		breadcrumbs[i] = 0;
	}
	breadcrumb_count = 0;
	return;
}


char* serialize() {
	char* baggage = malloc(64*sizeof(char));

	#define COPY(src, offset, len) {for(int i=0; i<len; i++) {baggage[i+offset] = src[i];} }
	COPY(service_addr, 0, 32);
	COPY(service_port, 32, 32);
	
	return (char*)baggage;
}

void deserialize(char* baggage) {
	// printf("pre-deser %d %d %u %d %u %d\n", parent_node_id, parent_span_id, parent_addr, parent_port, root_addr, root_port);
	
	// printf("%s\n", baggage);
	#define decode(dst, offset, len) { for(int i=0; i<len; i++) {dst[i] = baggage[i + offset];} }
	char* parent_addr;
	char* parent_port;
	decode(parent_addr, 0, 32);
	decode(parent_port, 32, 32);

	// trace_add_breadcrumb((AgentAddress*)whole_ip(parent_addr, parent_port));

	return;
}

void Lock(char* l) {
	while(!__sync_bool_compare_and_swap(l, '0', '1')) {
		sched_yield();
	}
	return;
}

void Unlock(char* l) {
	__sync_bool_compare_and_swap(l, '1', '0');
	return;
}

char* whole_ip(char* addr_, char* port_) {
	char* res = malloc(sizeof(char)*32);
	memset(res, 0, sizeof(char)*32);
	strcpy(res, addr_);
	strcat(res, ":");
	strcat(res, port_);
	return res;
}