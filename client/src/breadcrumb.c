#include <stdio.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <assert.h>
#include <string.h>

#include "breadcrumb.h"

char* breadcrumbs_get_fname(char* dst1, char* dst2) {
    char* name = malloc(sizeof(char)*64);
    memset(name, 0, sizeof(char)*64);
    strcpy(name, dst1);
    strcat(name, dst2);
    return name;
}

Breadcrumbs breadcrumbs_init(const char* name, size_t capacity) {
	Breadcrumbs b;
	b.name = name;
    b.queue = queue2_init(breadcrumbs_get_fname("/dev/shm/breadcrumbs_queue_", name), sizeof(Breadcrumb), capacity);
	return b;
}

void _breadcrumb_set_addr(Breadcrumb* crumb, const char* addr) {
    size_t addr_len = strlen(addr);
    if (addr_len > BREADCRUMB_MAX_SIZE-1) {
        addr_len = BREADCRUMB_MAX_SIZE-1;
    }
    memcpy(crumb->addr, addr, addr_len);
    crumb->addr[addr_len] = '\0';
}

void breadcrumbs_add(Breadcrumbs* b, uint64_t trace_id, const char* addr) {
    Breadcrumb crumb;
    crumb.trace_id = trace_id;
    crumb.type = 0;
    _breadcrumb_set_addr(&crumb, addr);
    queue2_put_nonblocking(&b->queue, (char*) &crumb);
}

// Add a forward breadcrumb
void breadcrumbs_add_forward(Breadcrumbs* b, uint64_t trace_id, const char* addr) {
    Breadcrumb crumb;
    crumb.trace_id = trace_id;
    crumb.type = 1;
    _breadcrumb_set_addr(&crumb, addr);
    queue2_put_nonblocking(&b->queue, (char*) &crumb);
}
