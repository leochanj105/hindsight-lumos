#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <time.h>
#include "assert.h"

#include "buffer.h"
#include "tracer2.h"
#include <pthread.h>

void test_buffer_simple() {
	Buffer b = buffer_create();
	assert(b.id == -1);
	assert(b.remaining == 0);
	assert(b.ptr == 0);

	printf("test_buffer_simple passed\n");
}

void test_buffer_write() {
	int buf_id = 5;
	size_t buf_size = 21;
	char buf[buf_size];

	Buffer b = buffer_create();
	b.id = buf_id;
	b.ptr = buf;
	b.remaining = buf_size;

	char* dst;
	size_t dst_size;
	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == buf);
	assert(dst_size == 5);
	assert(b.id == buf_id);
	assert(b.remaining == 16);
	assert(b.ptr == (buf + 5));
	assert(!buffer_is_full(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 5));
	assert(dst_size == 5);
	assert(b.id == buf_id);
	assert(b.remaining == 11);
	assert(b.ptr == (buf + 10));
	assert(!buffer_is_full(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 10));
	assert(dst_size == 5);
	assert(b.id == buf_id);
	assert(b.remaining == 6);
	assert(b.ptr == (buf + 15));
	assert(!buffer_is_full(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 15));
	assert(dst_size == 5);
	assert(b.id == buf_id);
	assert(b.remaining == 1);
	assert(b.ptr == (buf + 20));
	assert(!buffer_is_full(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 20));
	assert(dst_size == 1);
	assert(b.id == buf_id);
	assert(b.remaining == 0);
	assert(b.ptr == (buf + 21));
	assert(buffer_is_full(&b));

	buffer_write(&b, 5, &dst, &dst_size);

	assert(dst == (buf + 21));
	assert(dst_size == 0);
	assert(b.id == buf_id);
	assert(b.remaining == 0);
	assert(b.ptr == (buf + 21));
	assert(buffer_is_full(&b));

	printf("test_buffer_write passed\n");	
}

void test_bufmanager() {
	BufManager mgr = bufmanager_init("test_bufmanager", 10, 100);


	queue_put(mgr.available, 7);

	Buffer buf = buffer_create();
	bufmanager_acquire(&mgr, &buf);
	assert(buf.id == 7);
	buffer_clear(&buf);


	queue_put(mgr.available, 11);
	bufmanager_acquire(&mgr, &buf);
	assert(buf.id == 11);
	buffer_clear(&buf);

	for (unsigned i = 0; i < 100; i++) {
		queue_put(mgr.available, i);
		bufmanager_acquire(&mgr, &buf);
		assert(buf.id == i);
		buffer_clear(&buf);
	}

	for (unsigned i = 0; i < 100; i+=2) {
		queue_put(mgr.available, i);
		queue_put(mgr.available, i+1);
		bufmanager_acquire(&mgr, &buf);
		assert(buf.id == i);
		buffer_clear(&buf);
		bufmanager_acquire(&mgr, &buf);
		assert(buf.id == i+1);
		buffer_clear(&buf);
	}

	for (unsigned i = 0; i < 100; i+=2) {
		queue_put(mgr.available, i+1);
		queue_put(mgr.available, i);
		bufmanager_acquire(&mgr, &buf);
		assert(buf.id == i+1);
		buffer_clear(&buf);
		bufmanager_acquire(&mgr, &buf);
		assert(buf.id == i);
		buffer_clear(&buf);
	}

	printf("test_bufmanager passed\n");	
}

int main(int argc, char const *argv[])
{
	printf("Hello world!\n");
	test_buffer_simple();
	test_buffer_write();
	test_bufmanager();
	return 0;
}