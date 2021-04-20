#include "tracestate.h"

#include <assert.h>
#include <string.h>

void tracestate_begin(TraceState* trace, BufManager* mgr, long long trace_id) {
	if (!trace->active) {
		bufmanager_return(mgr, &trace->buffer);
	}
	trace->active = true;
	trace->buf_count = 1;
	trace->trace_id = trace_id;
	bufmanager_acquire(mgr, &trace->buffer);
}

void tracestate_end(TraceState* trace, BufManager* mgr) {
	if (!trace->active) return;
	bufmanager_return(mgr, &trace->buffer);
	trace->active = false;
	trace->buf_count = 0;
	trace->trace_id = 0;
}

void tracestate_write_data(TraceState* trace, 
                           BufManager* mgr,
	                       size_t write_size, 
	                       char** dst, 
	                       size_t* dst_size) {

	// Common case: do the write and exit
	buffer_write(&trace->buffer, write_size, dst, dst_size);
	if (*dst_size > 0) return;

	// Buffer is full, must swap
	bufmanager_return(mgr, &trace->buffer);
	bufmanager_acquire(mgr, &trace->buffer);

	// Retry the write
	buffer_write(&trace->buffer, write_size, dst, dst_size);

	// Implies we're getting zero-sized buffers
	assert(*dst_size > 0);
}

// Writes data to the trace; called by tracepoint
void tracestate_write(TraceState* trace, 
                      BufManager* mgr,
                      char* buf,
                      size_t buf_size) {
	char* dst;
	size_t dst_size;

	while (buf_size > 0) {
		// Try to write everything
		tracestate_write_data(trace, mgr, buf_size, &dst, &dst_size);

		// Write what we're allowed
		memcpy((void*) dst, (void*) buf, dst_size);

		buf += dst_size;
		buf_size -= dst_size;
	}
}