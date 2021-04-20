#ifndef _HINDSIGHT_TRACESTATE_H_
#define _HINDSIGHT_TRACESTATE_H_

#include <stdbool.h>

#include "buffer.h"

// TraceState represents an active, ongoing trace in the current process
typedef struct TraceState {
    bool active;
    int buf_count;
    long long trace_id;
    Buffer buffer;
} TraceState;

// Called when initializing the thread local tracestate
TraceState tracestate_init(BufManager* mgr);

// Starts a new trace state for the specified trace ID
// I think currently traceID is the only thing Hindsight should need
void tracestate_begin(TraceState* trace, BufManager* mgr, long long trace_id);

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




#endif // _HINDSIGHT_TRACESTATE_H_