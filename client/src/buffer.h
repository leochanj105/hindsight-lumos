#ifndef _HINDSIGHT_CLIENT_BUFFER_H_
#define _HINDSIGHT_CLIENT_BUFFER_H_

#include <stddef.h>
#include <stdbool.h>

// Points to a buffer allocated in shared memory
// Includes some metadata not stored in shared memory
typedef struct Buffer {
	int id; // Equivalent to the index of this buffer in the buffer pool
	size_t remaining; // Space remaining in the underlying buffer
	char* ptr; // Pointer to next available byte in buffer
} Buffer;

// Initializes a buffer with ID -1, nullptr, and 0 remaining
Buffer buffer_create_empty();

// Initializes a buffer with the provided ID, ptr, and remaining
Buffer buffer_create(int id, char* ptr, size_t size);

// Sets a buffer to ID -1, nullptr, and 0 remaining
void buffer_clear(Buffer* b);

// Sets the ID, ptr, and remaining size of a buffer
void buffer_update(Buffer* b, int id, char* ptr, size_t size);

// True if remaining is 0, false otherwise
bool buffer_isempty(Buffer* b);

// Returns remaining space in buffer
bool buffer_remaining(Buffer* b);

// Requests to write `size`-much data to the buffer.  The caller
// will receive a pointer in `dst` and will be responsible for actually
// writing the data to `dst`.  The caller can write up to `dst_size`
// data.
//
// If the caller requests more room than available (ie `size` > `remaining`)
// then `dst_size` will only be the remaining capacity, and the caller
// must acquire a new buffer to write the remaining data.
void buffer_write(Buffer* b, size_t size, char** dst, size_t* dst_size);

#endif // _HINDSIGHT_CLIENT_BUFFER_H_