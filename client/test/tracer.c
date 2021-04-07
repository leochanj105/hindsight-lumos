#include <stdio.h>
#include <stdlib.h>

#include "tracer.h"

int main(int argc, char const *argv[])
{
	trace_init(100);
	trace_begin(1,1,1);
	tracepoint(1,100);
	trace_add_breadcrumb("test");
	trace_end();


	printf("%d %d\n", pool[0], pool[2]);
	return 0;
}