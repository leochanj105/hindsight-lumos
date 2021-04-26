#ifndef _QUEUE_H_
#define _QUEUE_H_

#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>


#ifndef DEBUG
#define DEBUG 0
#endif

#define HEAD  0
#define TAIL  1
#define CAP   2
#define COUNT 3

#define IDX(x) x*2+4
#define AVL_IDX(x) x*2+5

typedef int* Queue;

Queue queue_init(const char* fname, int cap);

void queue_print(Queue queue);

void queue_put(Queue queue, int data);

int queue_get(Queue queue);




// Metadata at the start of the shared memory region of the queue
typedef struct QueueMetadata {
	bool initialized; // Set to true once everything is set up
	size_t capacity; // Capacity in number of elements
	size_t element_metadata_size; // Size of element metadata
	size_t element_size; // Size of one element content
	size_t element_total_size; // metadata + content
	__attribute__((aligned(64))) size_t head; // Index (not ptr) of the head of the queue
	__attribute__((aligned(64))) size_t tail; // Index (not ptr) of the tail of the queue
} QueueMetadata;

// Metadata at the start of each queue element
typedef struct QueueElementMetadata {
	int status; // 0=empty, 1=writing, 2=full, 3=reading
} QueueElementMetadata;

typedef struct Queue2 {
	// shmem pointers:
	QueueMetadata* meta; // Metadata of the queue, **within** the shmem region
	char* baseptr; // Baseptr of the shmem region
	char* queue; // Baseptr of the queue region, comes after the metadata
} Queue2;

// Return true if a shmem queue exists for the specified name
bool queue2_exists(const char* fname);

// Creates a new shm queue at the specified filename
Queue2 queue2_init(const char* fname, size_t element_size, size_t capacity);

// Uses an existing shm queue at the specified filename.
// Blocks until the file exists
Queue2 queue2_init_existing(const char* fname);

void queue2_put_blocking(Queue2* q, char* element);
void queue2_put_blocking_multi(Queue2* q, char* elements, size_t num_elements);
bool queue2_put_nonblocking(Queue2* q, char* element);
size_t queue2_put_nonblocking_multi(Queue2* q, char* elements, size_t num_elements);

void queue2_get_blocking(Queue2* q, char* dst_element);
bool queue2_get_nonblocking(Queue2* q, char* dst_element);
size_t queue2_get_nonblocking_multi(Queue2* q, char* elements, size_t max_elements);

#endif