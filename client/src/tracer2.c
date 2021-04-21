#include "tracer2.h"

#include "tracestate.h"

__thread TraceState threadlocal_tracestate = {false};

void* threadlocal() {
	printf("threadlocal tracestate is active: %d\n", threadlocal_tracestate.active);
	threadlocal_tracestate.active = true;
}