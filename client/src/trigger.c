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

void tail_init(int p) {
    tail = malloc(sizeof(TailCounter));
    tail->thres = (100-p)*10;
    for (int i=0; i<1000; i++) {
        tail->queue[i] = 0;
        if (i<tail->thres) tail->p[i] = 0;
    }
    tail->pos = 999;
    tail->p_curr = 0;
    return;
}


void tail_inject() {
    if (rand() % 100 < 10) {
        usleep(1000 * (20 + rand() % 10));
        printf("Latency injected\n");
    }

    return;
}

bool tail_latency(uint64_t req_id, int64_t latency) {
    // printf("P%d receiving latency %d\n", tail->thres, latency);
    tail->pos = (tail->pos+1) % 1000;
    int64_t evict = tail->queue[tail->pos];
    tail->queue[tail->pos] = latency;
    tail->pos++;

    // check evict, latency, and p_curr
    if (latency <= tail->p_curr) {
        if (evict > tail->p_curr) {
            int64_t max = 0;
            for (int i=0; i<1000; i++) {
                if (tail->queue[i] < tail->p_curr && tail->queue[i] > max) max = tail->queue[i];
            }
            for (int i=0; i<tail->thres; i++) 
                if (tail->p[i] == evict) {
                    tail->p[i] = max;
                    break;
                }
            
            tail->p_curr = tail->p[0];
            for (int i=0; i<tail->thres; i++) {
                if (tail->p_curr > tail->p[i]) tail->p_curr = tail->p[i];
            }
        }
        return false;
    } else {
        int rep_value = evict > tail->p_curr? evict : tail->p_curr;
        for(int i=0; i<tail->thres; i++) 
            if (tail->p[i] == rep_value) {
                tail->p[i] = latency;
                break;
            }
        
        tail->p_curr = tail->p[0];
        for (int i=0; i<tail->thres; i++)
            if (tail->p_curr > tail->p[i]) tail->p_curr = tail->p[i];

        printf("[TAIL LATENCY] Req: %ld Latency: %ld\n", req_id, latency);

        return true;
    }   
}

void tail_print() {
    printf("Curr P%d latency: %ld, pos: %d\n", tail->thres, tail->p_curr, tail->pos);
    for (int i=0; i<tail->thres; i++) printf("%ld ", tail->p[i]);
    printf("\n");
}

ExceptionHelper* exception;

void exception_init() {
    exception = malloc(sizeof(ExceptionHelper));
    exception->error_rate = 1;
    exception->stage = 0;
    exception->request_count = 0;
    exception->sampled_count = 0;
    exception->last_step = 0;
}

void exception_rate() {
    if (exception->last_step == 0) {
        exception->last_step = nanos();
        exception->stage = 1;
        return;
    }
    // if (exception->active == false) return;
    uint64_t dur = nanos() - exception->last_step;
    if (dur > 30000000000) {
        if (exception->error_rate < 10 && exception->stage < 10)
            exception->error_rate++;
        else if (exception->stage == 14 || exception->stage == 15) 
            exception->error_rate = 10; 
        else
            exception->error_rate = 1;
        
        exception->stage ++;
        exception->last_step = nanos();
        exception->request_count = 0;
        exception->sampled_count = 0;
        printf("Exception Rate Changed to %d\n", exception->error_rate);
    }
    return;
}

bool exception_throw(uint64_t req_id) {
    exception_rate();
    exception->request_count++;
    if (rand() % 100 < exception->error_rate) {
        printf("[EXCEPTION THROWN] Req %ld\n", req_id);
        return true;
    }
    return false;
}

bool exception_rate_limit() {
    if (exception->sampled_count * 100 <  exception->request_count) {
        exception->sampled_count++;
        return true;
    }
    return false;
}