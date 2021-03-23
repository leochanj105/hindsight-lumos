#define _GNU_SOURCE

#include "tracer.h"

// Global Structures

Pool pool;
SendQueue complete;
RecvQueue available;
SendQueue triggers;

// Thread Local Variables
__thread bool active;
__thread Header header;
__thread Buffer buffer;


// Tracer APIs

void flush(Metadata metadata){

}

void trace_init(){

}

void trace_begin(Metadata metadata){

}

void trace_end(){

}

void tracepoint(){

}

void trace_add_breadcrumb(AgentAddress breadcrumb){

}


Metadata trace_get_metadata(){
	return header.trace_md;
}

uint64_t trace_get_request_id(){
	return (uint64_t)0;
}

uint64_t trace_get_span_id(){
	return (uint64_t)0;
}

void trace_set_span_id(){

}

void trace_set_parent_span_id(){

}