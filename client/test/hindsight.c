#include <stdio.h>
#include <stdlib.h>

#include "buffer.h"
#include "tracer.h"

#define PROCESS_NAME "hs_integration_test"


void drain_forever() {
	BufManager bm = bufmanager_init_existing(PROCESS_NAME);
}


int main(int argc, char const *argv[])
{
	// TODO: capacity as argument
	// TODO: buffer size as argument
	// TODO: flag for agent/client

	if (argc > 0) {
		printf("Running as agent\n");
		drain_forever();
	} else {
		printf("Running as client\n");
	}



	return 0;
}