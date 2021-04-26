#include <stdio.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <assert.h>
#include <string.h>

#include "breadcrumb.h"
#include "common.h"

#define BREADCRUMBS_SHM_FILENAME(name) get_shm_fname(name, "breadcrumbs_queue")

Breadcrumbs breadcrumbs_init(const char* name, size_t capacity) {
    Breadcrumbs b;
    b.name = name;
    b.queue = queue2_init(BREADCRUMBS_SHM_FILENAME(name), sizeof(Breadcrumb), capacity);
    return b;
}

Breadcrumbs breadcrumbs_init_existing(const char* name) {
    Breadcrumbs b;
    b.name = name;
    b.queue = queue2_init_existing(BREADCRUMBS_SHM_FILENAME(name));
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
