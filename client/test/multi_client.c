#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <pthread.h>

#include "tracer.h"

#define num_threads 4
#define req_per_thread 100

pthread_barrier_t barrier;

void* client(void* arg) {
	int thread_num = (int*)arg;
	pthread_barrier_wait(&barrier);
	printf("thread num %d\n", thread_num);

	for (int i=thread_num*req_per_thread; i<(thread_num+1)*req_per_thread; i++) {
		printf("client request %d from thread %d\n", i, thread_num);

		TRACEBEGIN(i, 100*i, 200*i);
		TRACEPOINT(1, 100);
		trigger(i);
		TRACEEND();

		// sleep(1);
	}

	return NULL;
}

int main(int argc, char const *argv[])
{
	pthread_t threads[num_threads];
	pthread_barrier_init(&barrier, NULL, num_threads+1);

	trace_init(100);

	for (int i=0; i<num_threads; i++) {
		pthread_create(&threads[i], NULL, &client, (void*)i);
	}
	pthread_barrier_wait(&barrier);

	// pthread_barrier_destroy(&barrier);
	for (int i=0; i<num_threads; i++) {
		pthread_join(threads[i], NULL);
	}

	return 0;
}