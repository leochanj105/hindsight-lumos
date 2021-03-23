#include <stdbool.h>
#include <stdio.h>

#include "queue.h"

struct queue_t{
	uint64_t *c_queue;
	size_t head;
	size_t tail;
	size_t max;
	bool isFull;
};

queue_handle_t queue_init(int cap){
	queue_handle_t queue = malloc(sizeof(queue_t));
	queue->max = cap;
	printf("%d\n", cap);
	queue->c_queue = malloc(cap*sizeof(uint64_t));
	queue->head = 0;
	queue->tail = 0;
	queue->isFull = false;

	return queue;
}

size_t queue_size(queue_handle_t queue){
	return queue->max;
}

void queue_free(queue_handle_t queue){

}

void queue_reset(queue_handle_t queue){

}

void queue_empty(queue_handle_t queue){

}

void push(queue_handle_t queue){

}

void pop(queue_handle_t queue){

}