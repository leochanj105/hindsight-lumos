#ifndef _HINDSIGHT_TRACESTATE_H_
#define _HINDSIGHT_TRACESTATE_H_

#include <stdbool.h>

#include "buffer.h"

// TraceHeader represents the header data that Hindsight inserts at the start of every buffer
// It could include stuff like span IDs, but I'm not sure that's necessary in the header
typedef struct TraceHeader {
    uint64_t trace_id;
    uint64_t timestamp;
    short buffer_number;
    short null_buffer_count;
} TraceHeader;

// TraceState represents an active, ongoing trace in the current process
typedef struct TraceState {
    bool active;
    TraceHeader header; // The current trace header. Gets written to every buffer.
    Buffer buffer; // The current active buffer.
} TraceState;

// TraceState can also be initialized to {false}
TraceState tracestate_create();

// Starts a new trace state for the specified trace ID
// I think currently traceID is the only thing Hindsight should need
void tracestate_begin(TraceState* trace, BufManager* mgr, uint64_t trace_id);

// Ends the current trace state, flushes the buffer
void tracestate_end(TraceState* trace, BufManager* mgr);

// Acquires a buffer to write to, that will be in the trace
void tracestate_write_data(TraceState* trace, 
                           BufManager* mgr,
                           size_t write_size, 
                           char** dst, 
                           size_t* dst_size);

// Writes a buffer to the trace; called by tracepoint
void tracestate_write(TraceState* trace,
                      BufManager* mgr,
                      char* buf,
                      size_t buf_size);

// Attempts to one-shot write buffer
bool tracestate_try_write(TraceState* trace,
                          char* buf,
                          size_t buf_size);




#endif // _HINDSIGHT_TRACESTATE_H_