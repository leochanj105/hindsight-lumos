#ifndef _QUEUE_H_
#define _QUEUE_H_

#include <stdlib.h>
#include <stdint.h>

typedef struct queue_t queue_t;

typedef queue_t* queue_handle_t;

queue_handle_t queue_init(int cap);

size_t queue_size(queue_handle_t queue);

void queue_free(queue_handle_t queue);

void queue_reset(queue_handle_t queue);

void queue_empty(queue_handle_t queue);

void push(queue_handle_t queue);

void pop(queue_handle_t queue);


#endif