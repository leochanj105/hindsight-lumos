#ifndef _HINDSIGHT_CLIENT_COMMON_H_
#define _HINDSIGHT_CLIENT_COMMON_H_

#include <stdint.h>

char* get_shm_fname(char* dst1, char* dst2);

uint64_t nanos();

#endif // _HINDSIGHT_CLIENT_COMMON_H_