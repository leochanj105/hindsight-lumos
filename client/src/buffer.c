
#include <stdio.h>
#include "buffer.h"

Buffer buffer_create_empty() {
	Buffer b;
	buffer_clear(&b);
	return b;
}

Buffer buffer_create(int id, char* ptr, size_t remaining) {
	Buffer b;
	buffer_update(&b, id, ptr, remaining);
	return b;
}

void buffer_clear(Buffer* b) {
	buffer_update(b, -1, 0, 0);
}

void buffer_update(Buffer* b, int id, char* ptr, size_t remaining) {
	b->id = id;
	b->ptr = ptr;
	b->remaining = remaining;
}

int buffer_isempty(Buffer* b) {
	return b->remaining == 0;
}

void buffer_write(Buffer* b, size_t size, char** dst, size_t* dst_size) {
	if (b->remaining < size) size = b->remaining;

	*dst_size = size;
	*dst = b->ptr;

	b->remaining -= size;
	b->ptr += size;
}