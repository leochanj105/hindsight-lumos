#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#include "tracer.h"

int main(int argc, char const *argv[])
{
	trace_init("data");
	trace_begin(1000,1001,1002);
	tracepoint(10000,100);
	trace_add_breadcrumb("breadcrumb_test");
	trace_end();

	return 0;
}