#ifndef _QUEUE_H_
#define _QUEUE_H_

#include <stdlib.h>
#include <stdint.h>

#ifndef DEBUG
#define DEBUG 0
#endif

typedef struct queue_t queue_t;

typedef queue_t* Queue;

Queue queue_init(int cap);

void queue_print(Queue queue);

void queue_put(Queue queue, int data);

int queue_get(Queue queue);


#endif