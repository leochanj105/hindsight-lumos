#include <stdio.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <assert.h>
#include <string.h>

#include "trigger.h"
#include "common.h"

#define TRIGGERS_SHM_FILENAME(name) get_shm_fname(name, "triggers_queue")

Triggers triggers_init(const char* name, size_t capacity) {
    Triggers t;
    t.name = name;
    t.queue = queue_init(TRIGGERS_SHM_FILENAME(name), sizeof(Trigger), capacity);
    return t;
}

Triggers triggers_init_existing(const char* name) {
    Triggers t;
    t.name = name;
    t.queue = queue_init_existing(TRIGGERS_SHM_FILENAME(name));
    return t;
}

void triggers_fire(Triggers* t, int trigger_id, uint64_t trace_id) {
    Trigger trigger = {trigger_id, trace_id};
    queue_put_nonblocking(&t->queue, (char*) &trigger);
}

TailCounter* tail;

void tail_init() {
    tail = malloc(sizeof(TailCounter));
    for (int i=0; i<1000; i++) {
        tail->queue[i] = 0;
        if (i<10) tail->p99[i] = 0;
    }
    tail->pos = 999;
    tail->p99_curr = 0;
    return;
}

bool tail_p99(int latency) {
    printf("P99 receiving latency %d\n", latency);
    tail->pos = (tail->pos+1) % 1000;
    int evict = tail->queue[tail->pos];
    tail->queue[tail->pos] = latency;

    // check evict, latency, and p99_curr
    if (latency <= tail->p99_curr) {
        if (evict > tail->p99_curr) {
            int max = 0;
            for (int i=0; i<1000; i++) {
                if (tail->queue[i] < tail->p99_curr && tail->queue[i] > max) max = tail->queue[i];
            }
            for (int i=0; i<10; i++) 
                if (tail->p99[i] == evict) {
                    tail->p99[i] = max;
                    break;
                }
            
            tail->p99_curr = tail->p99[0];
            for (int i=0; i<10; i++) {
                if (tail->p99_curr > tail->p99[i]) tail->p99_curr = tail->p99[i];
            }
        }
        return false;
    } else {
        int rep_value = evict > tail->p99_curr? evict : tail->p99_curr;
        for(int i=0; i<10; i++) 
            if (tail->p99[i] == rep_value) {
                tail->p99[i] = latency;
                break;
            }
        
        tail->p99_curr = tail->p99[0];
        for (int i=0; i<10; i++)
            if (tail->p99_curr > tail->p99[i]) tail->p99_curr = tail->p99[i];

        return true;
    }   
}

void tail_print() {
    printf("Curr P99 latency: %d, pos: %d\n", tail->p99_curr, tail->pos);
    for (int i=0; i<10; i++) printf("%d ", tail->queue[i]);
    printf("\n");
    for (int i=0; i<10; i++) printf("%d ", tail->p99[i]);
    printf("\n");
}