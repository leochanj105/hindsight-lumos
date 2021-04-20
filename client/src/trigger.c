#include "trigger.h"


TriggerManager triggermanager_init(const char* name,
                                   size_t capacity) {
	TriggerManager mgr;
	mgr.name = name;
	mgr.triggers = queue_init(get_fname("/dev/shm/triggers_queue_", name), capacity);
	return mgr;
}


// For now, we are just sen
void triggermanager_trigger(TriggerManager* mgr, Trigger trigger) {
    // TODO: after queue refactor, enqueue into triggers

    // Lock(trigger_lock);
    // queue_put(triggers->queue, (int)(request_id_ >> 32));
    // queue_put(triggers->queue, (int)(request_id_ & 0xffffffff));
    // Unlock(trigger_lock);
    
}