#include <stdio.h>
#include "buffer.h"

#include <fcntl.h>
#include <sys/mman.h>
#include <assert.h>

char* bufmanager_get_fname(char* dst1, char* dst2) {
	char* name = malloc(sizeof(char)*64);
	memset(name, 0, sizeof(char)*64);
	strcpy(name, dst1);
	strcat(name, dst2);
	return name;
}

char* bufmanager_pool_init(const char* fname, size_t fsize) {
	void* shm;
	
	int fd = open(fname, O_RDWR | O_CREAT, 0666);
	assert(fd >= 0);

	int i = ftruncate(fd, fsize);
	assert(i == 0);

	shm = mmap(NULL, fsize, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
	assert(shm != MAP_FAILED);
	close(fd);

	memset(shm, 0, fsize);

	return (char*) shm;
}

BufManager bufmanager_init(const char* name,
                           size_t capacity,
                           size_t buffer_size) {
    BufManager m;
    m.name = name;

    size_t pool_size = capacity * buffer_size; // not *sizeof(int) we are storing bytes not ints...
    m.pool = bufmanager_pool_init(bufmanager_get_fname("/dev/shm/pool_", name), pool_size);
    m.capacity = capacity;
    m.buffer_size = buffer_size;


    m.available = queue2_init(bufmanager_get_fname("/dev/shm/available_queue_", name), sizeof(AvailableBuffer), capacity);
    m.complete = queue2_init(bufmanager_get_fname("/dev/shm/complete_queue_", name), sizeof(CompleteBuffer), capacity);

    m.null_buffer = (char*) malloc(buffer_size);
    return m;
}

void bufmanager_acquire(BufManager* mgr, Buffer* dst) {
    // Shouldn't be acquiring into a buffer that hasn't been released
    assert(!buffer_is_valid(dst));

    AvailableBuffer av = {-1};
    if (queue2_get_nonblocking(&mgr->available, &av)) {
    	dst->id = av.buffer_id;
    	dst->remaining = mgr->buffer_size;
        dst->ptr = mgr->pool + (av.buffer_id * mgr->buffer_size);
    } else {
    	dst->id = -2;
    	dst->remaining = mgr->buffer_size;
    	dst->ptr = mgr->null_buffer;
    }
}

void bufmanager_return(BufManager* mgr, uint64_t trace_id, Buffer* dst) {
	// No asserts; allowed to return an invalid buffer
	if (dst->id >= 0) {
		CompleteBuffer b = {trace_id, dst->id};
		queue2_put_blocking(&mgr->complete, &b);
	}
	buffer_clear(dst);
}


Buffer buffer_create() {
	Buffer b;
	buffer_clear(&b);
	return b;
}

void buffer_clear(Buffer* b) {
	b->id = -1;
	b->ptr = 0;
	b->remaining = 0;
}

bool buffer_is_full(Buffer* b) {
	return b->remaining == 0;
}

bool buffer_remaining(Buffer* b) {
	return b->remaining;
}

bool buffer_is_valid(Buffer* b) {
	return b->id >= 0;
}

void buffer_write(Buffer* b, size_t size, char** dst, size_t* dst_size) {
	if (b->remaining < size) size = b->remaining;

	*dst_size = size;
	*dst = b->ptr;

	b->remaining -= size;
	b->ptr += size;
}