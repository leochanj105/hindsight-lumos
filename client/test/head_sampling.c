#include <stdio.h>
#include <stdlib.h>
#include <time.h>

#include "tracer.h"

int main(int argc, char const *argv[]) {
	hindsight_init("sampling");
	char hbuf[100];
	while(true) {
		for(int i=0; i<=1000000; i+=100000) {
			// hindsight_begin_sampling(i);
		  	hindsight_begin(i);
		  	hindsight_tracepoint(hbuf, 1000);
			hindsight_end();
		}
	}
}