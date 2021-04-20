#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <time.h>
#include "assert.h"

#include "buffer.h"

void test_buffer_simple() {
	int buf_id = 5;
	size_t buf_size = 21;
	char buf[buf_size];

	Buffer b = buffer_create_empty();
	assert(b.id == -1);
	assert(b.remaining == 0);
	assert(b.ptr == 0);

	buffer_update(&b, buf_id, buf, buf_size);

	assert(b.id == buf_id);
	assert(b.remaining == 21);
	assert(b.ptr == buf);

	printf("test_buffer_simple passed\n");
}

void test_buffer_write() {
	int buf_id = 5;
	size_t buf_size = 21;
	char buf[buf_size];

	Buffer b = buffer_create(buf_id, buf, buf_size);
	assert(b.id == buf_id);
	assert(b.remaining == 21);
	assert(b.ptr == buf);

	char* dst;
	size_t dst_size;
	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == buf);
	assert(dst_size == 5);
	assert(b.id == buf_id);
	assert(b.remaining == 16);
	assert(b.ptr == (buf + 5));
	assert(!buffer_isempty(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 5));
	assert(dst_size == 5);
	assert(b.id == buf_id);
	assert(b.remaining == 11);
	assert(b.ptr == (buf + 10));
	assert(!buffer_isempty(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 10));
	assert(dst_size == 5);
	assert(b.id == buf_id);
	assert(b.remaining == 6);
	assert(b.ptr == (buf + 15));
	assert(!buffer_isempty(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 15));
	assert(dst_size == 5);
	assert(b.id == buf_id);
	assert(b.remaining == 1);
	assert(b.ptr == (buf + 20));
	assert(!buffer_isempty(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 20));
	assert(dst_size == 1);
	assert(b.id == buf_id);
	assert(b.remaining == 0);
	assert(b.ptr == (buf + 21));
	assert(buffer_isempty(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 21));
	assert(dst_size == 0);
	assert(b.id == buf_id);
	assert(b.remaining == 0);
	assert(b.ptr == (buf + 21));
	assert(buffer_isempty(&b));

	printf("test_buffer_write passed\n");	
}

int main(int argc, char const *argv[])
{
	printf("Hello world!\n");
	test_buffer_simple();
	test_buffer_write();
	return 0;
}