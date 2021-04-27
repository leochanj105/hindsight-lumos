#include "tracestate.h"

#include <assert.h>
#include <string.h>
#include <time.h>

TraceState tracestate_create() {
	TraceState trace = {false};
	return trace;
}

// Write the header to the current buffer
void tracestate_write_header(TraceState* trace) {
	char* dst;
	size_t dst_size;
	buffer_write(&trace->buffer, sizeof(TraceHeader), &dst, &dst_size);

	// write_header should only be called on a fresh buffer.
	assert(dst_size == sizeof(TraceHeader));

	// Write the header
	*((TraceHeader*) dst) = trace->header;
}

time_t tracestate_get_time() {
	struct timespec ts;
	timespec_get(&ts, TIME_UTC);
	return ts.tv_sec * 1000000000 + ts.tv_nsec;	
}

void tracestate_begin(TraceState* trace, BufManager* mgr, uint64_t trace_id) {
	// If we were previously active, return the buffer
	if (trace->active) {
		bufmanager_return(mgr, trace->header.trace_id, &trace->buffer);
	}
	buffer_clear(&trace->buffer);
	trace->active = true;

	// Set the new header
	trace->header.trace_id = trace_id;
	trace->header.buffer_number = 0;
	trace->header.null_buffer_count = 0;
	trace->header.timestamp = tracestate_get_time();

	// Acquire a fresh buffer and write the header
	bufmanager_acquire(mgr, &trace->buffer);
	tracestate_write_header(trace);
}

void tracestate_end(TraceState* trace, BufManager* mgr) {
	if (!trace->active) return;

	// Return the current buffer
	bufmanager_return(mgr, trace->header.trace_id, &trace->buffer);
	trace->active = false;

	// Clear the header
	trace->header.buffer_number = 0;
	trace->header.trace_id = 0;
	trace->header.null_buffer_count = 0;
}

void tracestate_write_data(TraceState* trace, 
                           BufManager* mgr,
	                       size_t write_size, 
	                       char** dst, 
	                       size_t* dst_size) {
	if (!trace->active) return;

	// Common case: there's room in the current buffer. Write and return.
	buffer_write(&trace->buffer, write_size, dst, dst_size);
	if (*dst_size != 0) return;

	// Buffer is full, return old buffer
	bufmanager_return(mgr, trace->header.trace_id, &trace->buffer);

	// Acquire new buffer and write header
	bufmanager_acquire(mgr, &trace->buffer);
	trace->header.buffer_number++;
	trace->header.timestamp = tracestate_get_time();
	if (trace->buffer.ptr == mgr->null_buffer) {
		// TODO: probably shouldn't be implemented like this
		trace->header.null_buffer_count++;
	}
	tracestate_write_header(trace);

	// Retry the write
	buffer_write(&trace->buffer, write_size, dst, dst_size);

	// Implies we're getting buffers with 0 capacity
	assert(*dst_size > 0);
}

// Writes data to the trace; called by tracepoint
void tracestate_write(TraceState* trace, 
                      BufManager* mgr,
                      char* buf,
                      size_t buf_size) {
	if (!trace->active) return;
	char* dst;
	size_t dst_size;

	while (buf_size != 0) {
		// Try to write everything
		tracestate_write_data(trace, mgr, buf_size, &dst, &dst_size);

		// Write what we're allowed
		memcpy((void*) dst, (void*) buf, dst_size);

		buf += dst_size;
		buf_size -= dst_size;
	}
}