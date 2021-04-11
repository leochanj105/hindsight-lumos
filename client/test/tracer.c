#include <stdio.h>
#include <stdlib.h>

#include "tracer.h"

int main(int argc, char const *argv[])
{
	trace_init(100);
	trace_begin(1000,1001,1002);
	tracepoint(1,100);
	trace_add_breadcrumb("test");
	trace_end();

	return 0;
}