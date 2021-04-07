#include <sys/mman.h>
#include <fcntl.h>
#include <assert.h>
#include <unistd.h>
#include <sys/types.h>
#include <string.h>

#include "memory.h"

int buffer_counter;

void trigger(uint64_t trigger_id){

}

int acquire(){
	buffer_counter++;
	return buffer_counter - 1;
}

void release(int buffer){

}

void* mem_init(const char* fname, size_t fsize) {
	void* shm;
	
	int fd = open(fname, O_RDWR | O_CREAT, 0666);
	assert(fd >= 0);

	// int isExist = isFileExist(new_fname);

	int i = ftruncate(fd, fsize);
	assert(i == 0);

	shm = mmap(NULL, fsize, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
	assert(shm != MAP_FAILED);	
	close(fd);

	// if(isEmpty || !isExist)
	memset(shm, 0, fsize);

	return shm;
}

bool isFileExist(const char* fname) {
	return (access(fname, F_OK) != -1);
}

void write_buffer(void* data, size_t offset) {

}