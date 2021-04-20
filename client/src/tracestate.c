#include "tracestate.h"

#include <assert.h>
#include <string.h>

// Write the header to the current buffer
void write_header(TraceState* trace) {
	char* dst;
	size_t dst_size;
	buffer_write(&trace->buffer, sizeof(TraceHeader), &dst, &dst_size);

	// write_header should only be called on a fresh buffer.
	assert(dst_size == sizeof(TraceHeader));

	// Write the header
	*((TraceHeader*) dst) = trace->header;
}

void tracestate_begin(TraceState* trace, BufManager* mgr, long long trace_id) {
	// If we were previously active, return the buffer
	if (trace->active) {
		bufmanager_return(mgr, &trace->buffer);
	}
	trace->active = true;

	// Set the new header
	trace->header.trace_id = trace_id;
	trace->header.buffer_number = 0;

	// Acquire a fresh buffer and write the header
	bufmanager_acquire(mgr, &trace->buffer);
	write_header(trace);
}

void tracestate_end(TraceState* trace, BufManager* mgr) {
	if (!trace->active) return;

	// Return the current buffer
	bufmanager_return(mgr, &trace->buffer);
	trace->active = false;

	// Clear the header
	trace->header.buffer_number = 0;
	trace->header.trace_id = 0;
}

void tracestate_write_data(TraceState* trace, 
                           BufManager* mgr,
	                       size_t write_size, 
	                       char** dst, 
	                       size_t* dst_size) {

	// Common case: there's room in the current buffer. Write and return.
	buffer_write(&trace->buffer, write_size, dst, dst_size);
	if (*dst_size > 0) return;

	// Buffer is full, must swap
	bufmanager_return(mgr, &trace->buffer);
	bufmanager_acquire(mgr, &trace->buffer);

	// Write the trace header
	write_header(trace);

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