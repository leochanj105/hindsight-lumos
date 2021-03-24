#include <stdbool.h>
#include <stdio.h>

#include "queue.h"

struct queue_t{
	int *buffer;
	int head;
	int tail;
	int cap;
	int count;
	bool isFull;
	bool isEmpty;
};

Queue queue_init(int cap){
	// TODO: queue might be defined by agent, change to mmap here
	Queue queue = malloc(sizeof(queue_t));
	queue->cap = cap;
	queue->buffer = malloc(cap*sizeof(int));
	queue->head = 0;
	queue->tail = 0;
	queue->isFull = false;
	queue->isEmpty = false;

	return queue;
}


void queue_print(Queue queue) {
	printf("head: %d tail: %d\n", queue->head, queue->tail);
	for (int i=0; i<queue->cap; i++) {
		printf("%d ", queue->buffer[i]);
	}
	printf("\n");
}

int get_head(Queue queue) {
	int head = __sync_fetch_and_add(&queue->head, 1);
	if (head == queue->cap) {
		int new_head = __sync_sub_and_fetch(&queue->head, queue->cap);
		#if(DEBUG)
			printf("mod head to %d\n", new_head);
		#endif
		queue->isFull = true;
	}
	return head % queue->cap;
}

void queue_put(Queue queue, int data){
	int head = get_head(queue);
	queue->buffer[head] = data;
	queue->count++;
	#if(DEBUG)
		printf("put %d at %d\n", data, head);
	#endif
	if (!queue->isEmpty) queue->isEmpty = true;
	return;
}

int get_tail(Queue queue) {
	if (queue->isEmpty == false) return -1;
	if (!queue->isFull && queue->tail >= queue->head) return -1;
	int tail = __sync_fetch_and_add(&queue->tail, 1);
	if (tail == queue->cap) {
		int new_tail = __sync_sub_and_fetch(&queue->tail, queue->cap);
		queue->isFull = false;
		#if(DEBUG)
			printf("mod tail to %d\n", new_tail);
		#endif
	}
	return tail % queue->cap;
}

int queue_get(Queue queue){
	int tail = get_tail(queue);
	if (tail == -1) return -1;
	int data = queue->buffer[queue->tail];
	#if(DEBUG)
		printf("get %d at %d\n", data, tail);
	#endif
	return data;
}