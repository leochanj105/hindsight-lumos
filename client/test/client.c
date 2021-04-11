#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#include "tracer.h"

int main(int argc, char const *argv[])
{
	trace_init(100);
	for (int i=0; i<200; i++) {
		printf("client request %d\n", i);
		trace_begin(i,1001,1002);
		tracepoint(1,100);
		trace_add_breadcrumb("test");
		trace_end();
		// sleep(1);	
	}
	// while(true) {
	// 	trace_begin(1000,1001,1002);
	// 	tracepoint(1,100);
	// 	trace_add_breadcrumb("test");
	// 	trace_end();
	// 	sleep(1);
	// }



	return 0;
}