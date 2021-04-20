#ifndef _HINDSIGHT_CLIENT_TRIGGER_H_
#define _HINDSIGHT_CLIENT_TRIGGER_H_

#include <stddef.h>
#include <stdbool.h>

#include "queue.h"

typedef struct TriggerManager {
    const char* name; // Name of this service

    Queue triggers; // Used to send triggers
} TriggerManager;

// For now, a trigger is just an ID and trace_id
typedef struct Trigger {
    int trigger_id; // The ID of the trigger that fired
    long long trace_id; // The trace that fired it
} Trigger;

// name is used for mapping to the appropriate shmem file
// capacity is used to decide queue size
TriggerManager triggermanager_init(const char* name,
                                   size_t capacity);

// For now, we are just sen
void triggermanager_trigger(TriggerManager* mgr, Trigger trigger);


#endif // _HINDSIGHT_CLIENT_TRIGGER_H_