package main

/*
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <assert.h>
#include <sched.h>
typedef struct QueueMetadata {
	bool initialized; // Set to true once everything is set up
	size_t capacity; // Capacity in number of elements
	size_t element_metadata_size; // Size of element metadata
	size_t element_size; // Size of one element content
	size_t element_total_size; // metadata + content
	__attribute__((aligned(64))) size_t head; // Index (not ptr) of the head of the queue
	__attribute__((aligned(64))) size_t tail; // Index (not ptr) of the tail of the queue
} QueueMetadata;

// Metadata at the start of each queue element
typedef struct QueueElementMetadata {
	int status; // 0=empty, 1=writing, 2=full, 3=reading
} QueueElementMetadata;

typedef struct Queue2 {
	// shmem pointers:
	QueueMetadata* meta; // Metadata of the queue, **within** the shmem region
	char* baseptr; // Baseptr of the shmem region
	char* queue; // Baseptr of the queue region, comes after the metadata
} Queue2;

Queue2 queue2_init_existing(const char* fname)  {
	Queue2 q;

	// Wait until the file exists
	while (access(fname, F_OK) != 0) {
		printf("%s does not exist, waiting...\n", fname);
		usleep(1000000);
	}
	
	// Open the file, get its length
	int fd = open(fname, O_RDWR, 0666);
	assert(fd >= 0);

	struct stat st;
	fstat(fd, &st);
	size_t shmem_size = st.st_size;

	void* shm = mmap(NULL, shmem_size, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
	assert(shm != MAP_FAILED);	
	close(fd);

	q.meta = (QueueMetadata*) shm;
	q.baseptr = (char*) shm;
	q.queue = q.baseptr + sizeof(QueueMetadata);

    while (!q.meta->initialized) {
    	printf("Waiting for initialization of %s...\n", fname);
    	usleep(1000000);
    }

    printf("Loaded existing queue ");
    printf("capacity=%ld ", q.meta->capacity);
    printf("element_size=%ld ", q.meta->element_size);
    printf("element_total_size=%ld ", q.meta->element_total_size);
    printf("at %s\n", fname);

	return q;

}
*/
import "C"

import (
	"fmt"
	"os"
	"syscall"
)




func main() {
	fmt.Println("Hello world!")

	fname := "/dev/shm/available_queue_test_tracestate"

	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("open file failed:", err)
	}
	fd := int(f.Fd())
	fmt.Println("opened ", fd)

	fi, err := f.Stat()
	if err != nil {
		fmt.Println("stat failed:",err)
	}
	fmt.Println("size is ", fi.Size())

	p, err := syscall.Mmap(fd, 0, int(fi.Size()), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)

	fmt.Printf("%T\n", p)

	q := C.queue2_init_existing(C.CString(fname))

	fmt.Println(q)

	fmt.Println(q.meta)

	// md := C.QueueMetadata(p)
}
