#include "agentapi.h"


HindsightAgentAPI* hindsight_agentapi_init(const char* servicename) {
	HindsightAgentAPI* api = malloc(sizeof(HindsightAgentAPI));
	api->mgr = bufmanager_init_existing(servicename);
	api->triggers = triggers_init_existing(servicename);
	api->breadcrumbs = breadcrumbs_init_existing(servicename);
	return api;
}