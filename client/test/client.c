#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#include "tracer.h"

int main(int argc, char const *argv[])
{
	trace_init("client");
	for (int i=0; i<200; i++) {
		int i=1;
		printf("client request %d\n", i);
		trace_begin(i,1001,1002);
		tracepoint(1,100);
		trace_add_breadcrumb("localhost:5050");
		trigger(i);
		trace_end();
		// sleep(1);	
	}

	return 0;
}