#ifndef _MEMORY_H_
#define _MEMORY_H_

#include <stdlib.h>
#include <inttypes.h>

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


void trigger(uint64_t trigger_id);

Buffer acquire();

void release(Buffer buffer);


#endif