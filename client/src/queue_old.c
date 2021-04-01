#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/mman.h>
#include <fcntl.h>
#include <assert.h>
#include <sched.h>

#include "queue.h"

struct queue_t{
	int head;
	int tail;
	int cap;
	int count;
	int *buf;
	int *buf_avail;
};

Queue queue_init(int cap){
	size_t fsize = 4*sizeof(int)+cap*sizeof(int)*2;
	Queue queue = malloc(fsize);

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
	while(__sync_val_compare_and_swap(&queue->buf_avail[queue->head % queue->cap], 0, 1) != 0) {
		sched_yield();
	}
	int head = __sync_fetch_and_add(&queue->head, 1);

	if (head == queue->cap) {
		int new_head = __sync_sub_and_fetch(&queue->head, queue->cap);
		#if(DEBUG)
			printf("mod head to %d\n", new_head);
		#endif
	}

	return head % queue->cap;
}

void queue_put(Queue queue, int data){
	int head = get_head(queue);
	queue->buf[head] = data;
	__sync_fetch_and_add(&queue->count, 1);
	queue->buf_avail[head] = 2;

	#if(DEBUG)
		printf("PUT %d AT %d\n", data, head);
	#endif

	return;
}

int get_tail(Queue queue) {
	while(__sync_val_compare_and_swap(&queue->buf_avail[queue->tail % queue->cap], 2, 0) != 2) {
		sched_yield();
	}
	int tail = __sync_fetch_and_add(&queue->tail, 1);

	if (tail == queue->cap) {
		int new_head = __sync_sub_and_fetch(&queue->tail, queue->cap);
		#if(DEBUG)
			printf("mod tail to %d\n", new_head);
		#endif
	}

	return tail % queue->cap;
}

int queue_get(Queue queue){
	int tail = get_tail(queue);
	int data = queue->buf[tail];
	queue->buf_avail[tail] = 0;
	#if(DEBUG)
		printf("GET %d AT %d\n", data, tail);
	#endif

	return data;
}