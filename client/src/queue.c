#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/mman.h>
#include <fcntl.h>
#include <assert.h>
#include <sched.h>

#include "queue.h"
#include "memory.h"

Queue queue_init(const char* fname, int cap){
	Queue queue;
	size_t fsize = 4*sizeof(int)+cap*sizeof(int)*2;

	int fd = open(fname, O_RDWR | O_CREAT, 0666);
	assert(fd >= 0);

	// int isExist = isFileExist(fname);

	int i = ftruncate(fd, fsize);
	assert(i == 0);

	queue = (Queue)mmap(NULL, fsize, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
	assert(queue != MAP_FAILED);	
	close(fd);

	memset(queue, 0, fsize);

	queue[HEAD] = 0;
	queue[TAIL] = 0;
	queue[CAP] = cap;
	queue[COUNT] = 0;

	return queue;
}

void queue_print(Queue queue) {
	printf("head: %d tail: %d\n", queue[HEAD], queue[TAIL]);
	for (int i=0; i<queue[CAP]; i++) {
		printf("%d %d\n", queue[IDX(i)], queue[AVL_IDX(i)]);
	}
	printf("\n");
}

int get_head(Queue queue) {
	while(__sync_val_compare_and_swap(queue+AVL_IDX(queue[HEAD]%queue[CAP]), 0, 1) != 0) {
		// #if(DEBUG)
		// 	printf("yielding for head %d %d %d %d %d %d\n", queue[HEAD], queue[TAIL], queue[CAP], queue[COUNT], queue[AVL_IDX(queue[HEAD]%queue[CAP])], queue[AVL_IDX(queue[TAIL]%queue[CAP])]);
		// #endif
		sched_yield();
	}
	int head = __sync_fetch_and_add(queue+HEAD, 1);

	if (head == queue[CAP]) {
		#if(DEBUG)
			int new_head = __sync_sub_and_fetch(queue+HEAD, queue[CAP]);
			printf("mod head to %d\n", new_head);
		#else
			__sync_sub_and_fetch(queue+HEAD, queue[CAP]);
		#endif
	}

	return head % queue[CAP];
}

void queue_put(Queue queue, int data){
	int head = get_head(queue);
	queue[IDX(head)] = data;
	__sync_fetch_and_add(queue + COUNT, 1);
	queue[AVL_IDX(head)] = 2;

	#if(DEBUG)
		printf("PUT %d AT %d\n", data, head);
	#endif

	return;
}

int get_tail(Queue queue) {
	while(__sync_val_compare_and_swap(queue + AVL_IDX(queue[TAIL] % queue[CAP]), 2, 3) != 2) {
		// #if(DEBUG)
		// 	printf("yielding for tail %d %d %d %d %d %d\n", queue[HEAD], queue[TAIL], queue[CAP], queue[COUNT], queue[AVL_IDX(queue[HEAD]%queue[CAP])], queue[AVL_IDX(queue[TAIL]%queue[CAP])]);
		// #endif
		sched_yield();
	}
	int tail = __sync_fetch_and_add(queue+TAIL, 1);

	if (tail == queue[CAP]) {
		#if(DEBUG)
			int new_tail = __sync_sub_and_fetch(queue+TAIL, queue[CAP]);
			printf("mod tail to %d\n", new_tail);
		#else
			__sync_sub_and_fetch(queue+TAIL, queue[CAP]);
		#endif

	}

	return tail % queue[CAP];
}

int queue_get(Queue queue){
	int tail = get_tail(queue);
	int data = queue[IDX(tail)];
	queue[AVL_IDX(tail)]=0;
	#if(DEBUG)
		printf("GET %d AT %d\n", data, tail);
	#endif

	return data;
}



Queue2 queue2_init(const char* fname, size_t element_size, size_t capacity) {
	Queue2 q;

	size_t element_metadata_size = sizeof(QueueElementMetadata);
	size_t element_total_size = element_size + element_metadata_size;

	size_t shmem_size = sizeof(QueueMetadata) + capacity * element_total_size;

	int fd = open(fname, O_RDWR | O_CREAT, 0666);
	assert(fd >= 0);

	// int isExist = isFileExist(fname);

	int i = ftruncate(fd, shmem_size);
	assert(i == 0);

	void* shm = mmap(NULL, shmem_size, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
	assert(shm != MAP_FAILED);	
	close(fd);

	memset(shm, 0, shmem_size);

	q.meta = (QueueMetadata*) shm;
	q.baseptr = (char*) shm;
	q.queue = q.baseptr + sizeof(QueueMetadata);

	q.meta->head = 0;
	q.meta->tail = 0;
	q.meta->capacity = capacity;
	q.meta->element_metadata_size = element_metadata_size;
	q.meta->element_size = element_size;
	q.meta->element_total_size = element_total_size;
	q.meta->initialized = true;

	return q;
}

Queue2 queue2_init_existing(const char* fname) {
	Queue2 q;


	return q;

}

char* queue2_ptr(Queue2* q, size_t index) {
	index = index % q->meta->capacity;
	return q->queue + (q->meta->element_total_size * index);
}

bool queue2_get_nonblocking(Queue2* q, char* element) {
	while (true) {
		// First, read the current head and tail values of the queue
		__sync_synchronize();
		size_t head = q->meta->head;
		size_t tail = q->meta->tail;

		// If the queue is currently empty, we can return
		int64_t delta = tail-head;
		if (delta <= 0) {
			return false;
		}

		// Try updating the head pointer; somebody else might have taken it
		if (!__sync_bool_compare_and_swap(&q->meta->head, head, head+1)) {
			continue;
		}

		// We got the slot.  Grab its pointer
		char* e_ptr = queue2_ptr(q, head);
		QueueElementMetadata* e_md = (QueueElementMetadata*) e_ptr;

		// It's possible a writer is still writing this element
		// Even though this should be a non-blocking call, we will block here :(
		int max_backoff = 100000; // 100ms
		int backoff = 10;
		while (!__sync_bool_compare_and_swap(&e_md->status, 2, 3)) {
			usleep(backoff);
			backoff *= 2;
			if (backoff > max_backoff) {
				backoff = max_backoff;
			}
		}

		// Read the element
		char* e_content = e_ptr + sizeof(QueueElementMetadata);
		memcpy(element, e_content, q->meta->element_size);

		// Update status, fail if somebody else touched it
		assert(__sync_bool_compare_and_swap(&e_md->status, 3, 0));

		return true;
	}
}

void queue2_get_blocking(Queue2* q, char* element) {
	// Only allowed to put if (tail-head) < capacity
	int max_backoff = 100000; // 100ms
	int backoff = 10;

	// Call non-blocking impl and backoff
	while (!queue2_get_nonblocking(q, element)) {
		usleep(backoff);
		backoff *= 2;
		if (backoff > max_backoff) {
			backoff = max_backoff;
		}
	}
}

bool queue2_put_nonblocking(Queue2* q, char* element) {
	while (true) {
		// First, read the current head and tail values of the queue
		__sync_synchronize();
		size_t tail = q->meta->tail;
		size_t head = q->meta->head;

		// If the queue is currently full, we can simply return
		int64_t delta = tail-head;
		if (delta >= q->meta->capacity) {
			return false;
		}

		// Try updating the tail pointer; somebody else might have taken it
		if (!__sync_bool_compare_and_swap(&q->meta->tail, tail, tail+1)) {
			continue;
		}

		// We got the slot.  Grab its pointer
		char* e_ptr = queue2_ptr(q, tail);
		QueueElementMetadata* e_md = (QueueElementMetadata*) e_ptr;

		// It's possible a reader is still reading this element
		// Even though this should be a non-blocking call, we will block here :(
		int max_backoff = 100000; // 100ms
		int backoff = 10;
		while (!__sync_bool_compare_and_swap(&e_md->status, 0, 1)) {
			usleep(backoff);
			backoff *= 2;
			if (backoff > max_backoff) {
				backoff = max_backoff;
			}
		}

		// Write the element
		char* e_content = e_ptr + sizeof(QueueElementMetadata);
		memcpy(e_content, element, q->meta->element_size);

		// Update status, fail if somebody else touched it
		assert(__sync_bool_compare_and_swap(&e_md->status, 1, 2));

		return true;
	}

}

void queue2_put_blocking(Queue2* q, char* element) {
	// Only allowed to put if (tail-head) < capacity
	int max_backoff = 100000; // 100ms
	int backoff = 10;

	// Call non-blocking impl and backoff
	while (!queue2_put_nonblocking(q, element)) {
		usleep(backoff);
		backoff *= 2;
		if (backoff > max_backoff) {
			backoff = max_backoff;
		}
	}
}



