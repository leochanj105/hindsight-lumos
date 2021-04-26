#include "common.h"

#include <stdio.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <assert.h>
#include <unistd.h>
#include <sys/stat.h>

char* get_shm_fname(char* dst1, char* dst2) {
	char* name = malloc(sizeof(char)*64);
	memset(name, 0, sizeof(char)*64);
	strcpy(name, "/dev/shm/");
	strcat(name, dst1);
	strcat(name, "__");
	strcat(name, dst2);
	return name;
}