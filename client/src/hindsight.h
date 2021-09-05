#ifndef _HINDSIGHT_HINDSIGHT_H_
#define _HINDSIGHT_HINDSIGHT_H_

#include <stdbool.h>
#include <inttypes.h>
#include <stdio.h>

#include "buffer.h"
#include "breadcrumb.h"
#include "trigger.h"
#include "tracestate.h"

#define HINDSIGHT_DEFAULT_CONFIG "/etc/hindsight_conf/default.conf"

#define TRIGGER_ID_HEAD_BASED_SAMPLING 1

/*
All of Hindsight's APIs can be invoked from tracestate.h

This file only does two extra things:
(1) it maintains a thread-local instance of TraceState
(2) it maintains a global instance of BufManager, Breadcrumbs,
    and Triggers, which use shared memory

In general, use the APIs in this file.  If you need multiple
instances of Hindsight, or if you don't want thread-local
trace states, then you can create and use instances directly.
*/

typedef struct HindsightConfig {
    size_t pool_capacity;
    size_t buffer_size;
    size_t breadcrumbs_capacity;
    size_t triggers_capacity;
    char* address; // max 32 bytes addr:port string
    int payload;
    float retroactive_sampling_percentage; // percentage of requests that we will sample; between 0 to 1; default 1
    float head_sampling_probability; // probability of automatically triggering for head-based sampling; between 0 to 1; default 0

    uint64_t _retroactive_sampling_threshold; // derived from retroactive_sampling_percentage
    uint64_t _head_sampling_threshold; // derived from head_sampling_probability
} HindsightConfig;

typedef struct Hindsight {
    HindsightConfig config;
    BufManager mgr;
    Breadcrumbs breadcrumbs;
    Triggers triggers;
} Hindsight;

// Single global instance of Hindsight state
extern Hindsight hindsight;
extern BufManager* mgr;

// Thread-local trace state
extern __thread TraceState hindsight_tracestate;

// Load a hindsight config from the specified file
HindsightConfig hindsight_load_config_file(const char* fname);

// Load a hindsight config for the specified service name; configs are located in /etc/hindsight_conf, e.g. loads /etc/hindsight_conf/{service_name}.conf
HindsightConfig hindsight_load_config(const char* service_name);

// Hindsight config defaults
HindsightConfig hindsight_default_config();

// Print the config to stdout
void hindsight_print_config(HindsightConfig* conf);

/*
This method must be called before any other Hindsight API is used,
to initialize Hindsight's global state

It will load the HindsightConfig from /etc/hindsight_conf/{service_name}.conf

If it is not called, Hindsight's shmem won't be set up, and you'll
get a segfault.
*/
void hindsight_init(const char* service_name);

/*
Same as hindsight_init, but uses the provided config.
*/
void hindsight_init_with_config(const char* service_name, HindsightConfig config);

// The current thread is beginning execution of the specified trace_id
void hindsight_begin(uint64_t trace_id);

// Call this if the trace has been head-sampled already
// Will ignore all sampling probabilities and always trace + trigger the trace
void hindsight_begin_sampled(uint64_t trace_id);

// The current thread has completed execution
void hindsight_end();

// Copies the provided data.
void hindsight_tracepoint(char* buf, size_t buf_size);

// Similar to hindsight_tracepoint, except instead of
// copying the data, returns a buffer to which the caller can
// write.
//   `write_size` - the requested size of the buffer to write to
//   `dst` - this method will store the buffer pointer here
//   `dst_size` - the size of the buffer stored in `dst`.  Not guaranteed
//                to be equal to write_size, in which case this method
//                should be invoked again to write the remaining data
void hindsight_tracepoint_write(size_t write_size, char** dst, size_t* dst_size);


// Report a breadcrumb from a previous node.  For now, addr is a literal host:port string
void hindsight_breadcrumb(const char* addr);

// Report a breadcrumb to a future node.  For now, addr is a literal host:port string
void hindsight_forward_breadcrumb(const char* addr);

// Fire a trigger.  For now, only report a trigger ID
void hindsight_trigger(int trigger_id);
void hindsight_trigger_manual(uint64_t trace_id, int trigger_id);

uint64_t hindsight_get_traceid();
char* hindsight_get_local_address();
bool hindsight_get_is_head_sampled();

int hindsight_null_buffer_count();

char* hindsight_serialize();

void hindsight_deserialize(char* baggage);

int hindsight_payload();

float hindsight_retroactive_sampling_percentage();

float hindsight_head_sampling_probability();

bool hindsight_is_active();
bool hindsight_is_recording();


#endif // _HINDSIGHT_HINDSIGHT_H_