#ifndef _HINDSIGHT_TRACER2_H_
#define _HINDSIGHT_TRACER2_H_

#include <stdbool.h>
#include <inttypes.h>
#include <stdio.h>

// Initialize Hindsight.  Must be called before other APIs are used.
void hindsight_init(const char* service_name);


void hindsight_begin(uint64_t trace_id);
void hindsight_end();

// Write this data directly to the buffer
void hindsight_tracepoint(char* buf, size_t buf_size);

// Request a buffer that the client app will write to
// `write_size` is the caller's requested buffer size
// after returning, `dst` will be a pointer to a buffer
// `dst` will have `dst_size` bytes available
// if `dst_size` < `write_size`, the caller should do
// a partial write then call `tracepoint_write` again
void hindsight_tracepoint_write(size_t write_size, char** dst, size_t* dst_size);

// Breadcrumbs are ipv4 addresses
typedef struct Breadcrumb {
  int addr[4];
  short port;
} Breadcrumb;

// Use ipv4 format struct or strings as done in current impl?
void hindsight_breadcrumb(Breadcrumb b); // TODO: argument type
void hindsight_forward_breadcrumb(Breadcrumb b); // TODO: argument type

void hindsight_trigger(int trigger_id); // TODO: argument types

// TODO: compatibility class with same method signature as Lei impl.

// TODO: config in separate file


#endif // _HINDSIGHT_TRACER2_H_