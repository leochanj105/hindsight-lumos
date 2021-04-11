#ifndef _QUEUE_H_
#define _QUEUE_H_

#include <stdlib.h>
#include <stdint.h>


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

#endif