#include <stdio.h>
#include <stdlib.h>

#include "hindsight.h"

int main(int argc, char const *argv[]) {
	hindsight_init("exception");
	char hbuf[100];

	printf("[EXCEPTION] hindsight\n");
	exception_init();
	for (int i=0; i<200; i++) 
	{
		hindsight_begin(i);
		hindsight_tracepoint(hbuf, 1000);
		hindsight_exception(100);
		hindsight_end();
		// sleep(1);
		
	}

	printf("[EXCEPTION] head based sampling\n");
	exception_init();
	for(int i=0; i<=1000000; i+=100000) {
		hindsight_begin_sampling(i);
	  	hindsight_tracepoint_sampling(hbuf, 1000);
		hindsight_trigger_sampling_exception(i);		  
		hindsight_end();
	}

}
