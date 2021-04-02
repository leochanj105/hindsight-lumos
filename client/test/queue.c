#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <time.h>

#include "queue.h"

#define num_threads 1
#define enqueue_per_thread 10
#define queue_cap 200

Queue queue;

pthread_barrier_t barrier;

void* get(void* arg) {
	// pthread_barrier_wait(&barrier);

	int count = 0;
	while(1) {
		count++;
		int data = queue_get(queue);
		for(int j=0; j<1; j++) {
			sched_yield();
		}
	}
	return NULL;
}

void* put(void* arg) {
	int thread_num = (int)arg;
	
	pthread_barrier_wait(&barrier);
	printf("thread %d\n", thread_num);

	for(int i=0; i<enqueue_per_thread; i++) {
		queue_put(queue, thread_num*enqueue_per_thread+i);
		printf("put %d\n", thread_num*enqueue_per_thread+i);
		for(int j=0; j<1; j++) {
			sched_yield();
		}
	}

	return NULL;
}


int main(int argc, char const *argv[])
{
	queue = queue_init(queue_cap);

	pthread_barrier_init(&barrier, NULL, num_threads);

	pthread_t threads[num_threads];
	// pthread_create(&threads[0], NULL, &get, (void *)0);
	for (int i=0; i<num_threads; i++) {
		pthread_create(&threads[i], NULL, &put, (void *)i);
	}

	pthread_barrier_destroy(&barrier);
	
	for (int i=0; i<num_threads; i++) {
		pthread_join(threads[i], NULL);
	}

	printf("Print queue content:\n");
	queue_print(queue);

	return 0;
}