#include <stdio.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <assert.h>
#include <string.h>

#include "trigger.h"

char* triggers_get_fname(char* dst1, char* dst2) {
    char* name = malloc(sizeof(char)*64);
    memset(name, 0, sizeof(char)*64);
    strcpy(name, dst1);
    strcat(name, dst2);
    return name;
}

Triggers triggers_init(const char* name, size_t capacity) {
    Triggers t;
    t.name = name;
    t.queue = queue2_init(triggers_get_fname("/dev/shm/triggers_queue_", name), sizeof(Trigger), capacity);
    return t;
}

Triggers triggers_init_existing(const char* name) {
    Triggers t;
    t.name = name;
    t.queue = queue2_init_existing(triggers_get_fname("/dev/shm/triggers_queue_", name));
    return t;
}

void triggers_fire(Triggers* t, int trigger_id, uint64_t trace_id) {
    Trigger trigger = {trigger_id, trace_id};
    queue2_put_nonblocking(&t->queue, (char*) &trigger);
}