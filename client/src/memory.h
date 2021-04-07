#ifndef _MEMORY_H_
#define _MEMORY_H_

#include <stdlib.h>
#include <stdbool.h>
#include <inttypes.h>

extern int buffer_counter;

void trigger(uint64_t trigger_id);

int acquire();

void release(int buffer);

void* mem_init(const char* fname, size_t fsize);

bool isFileExist(const char* fname);

void write_buffer(void* data, size_t offset);

#endif