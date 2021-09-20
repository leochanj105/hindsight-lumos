#include <stdio.h>
#include <stdlib.h>

#include "tracer.h"

int main(int argc, char const *argv[]) {
	hindsight_init("tail");
	char hbuf[100];

	printf("[TAIL LATENCY] hindsight\n");
	tail_init(99);
	for (int i=0; i<100; i++) 
	{
		hindsight_begin(i);
		hindsight_tracepoint(hbuf, 1000);
		hindsight_tail(i, 100);
		hindsight_end();
	}

	printf("[TAIL LATENCY] head based sampling\n");
	tail_init(99);
	for(int i=0; i<=1000000; i+=100000) {
		hindsight_begin_sampling(i);
	  	hindsight_tracepoint_sampling(hbuf, 1000);
		hindsight_trigger_sampling_tail(100, 0);
		hindsight_end();
	}
}
