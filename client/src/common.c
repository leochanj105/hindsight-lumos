#include "common.h"

#include <stdio.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <assert.h>
#include <unistd.h>
#include <sys/stat.h>
#include <time.h>


char* get_shm_fname(char* dst1, char* dst2) {
    char* name = malloc(sizeof(char)*64);
    memset(name, 0, sizeof(char)*64);
    strcpy(name, "/dev/shm/");
    strcat(name, dst1);
    strcat(name, "__");
    strcat(name, dst2);
    return name;
}

void truncate_string(char* dst, const char* src, size_t max_size) {
    size_t src_len = strlen(src);
    if (src_len > max_size-1) {
        src_len = max_size-1;
    }
    memcpy(dst, src, src_len);
    dst[src_len] = '\0';
}

uint64_t nanos() {
    struct timespec t;
    clock_gettime(CLOCK_MONOTONIC_RAW, &t);
    uint64_t nanos = t.tv_sec * 1000000000UL + t.tv_nsec;
    return nanos;
}