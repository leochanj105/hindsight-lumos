#include <stdio.h>
#include <stdlib.h>

#include "buffer.h"
#include "tracer.h"
#include "hindsight.h"
#include "agentapi.h"

#define PROCESS_NAME "hs_integration_test"
#define CAPACITY 1000
#define BUFFERSIZE 50


void drain_forever() {
	HindsightAgentAPI* api = hindsight_agentapi_init(PROCESS_NAME);
	// BufManager bm = bufmanager_init_existing(PROCESS_NAME);

	printf("Inited existing bufmanager\n");

	printf("Making %ld buffers available", api->mgr.meta->capacity);

	while (true)
		usleep(1000000);
}

void create_and_wait() {
	hindsight_init(PROCESS_NAME);
	// BufManager bm = bufmanager_init(PROCESS_NAME, CAPACITY, BUFFERSIZE);

	while (true)
		usleep(1000000);


}


int main(int argc, char const *argv[])
{
	// TODO: capacity as argument
	// TODO: buffer size as argument
	// TODO: flag for agent/client

	if (argc <= 1) {
		printf("Running as agent\n");
		drain_forever();
	} else {
		printf("Running as client\n");
		create_and_wait();
	}



	return 0;
}