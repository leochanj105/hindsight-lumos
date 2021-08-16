#include <stdio.h>
#include <stdlib.h>

#include "tracer.h"

int main(int argc, char const *argv[]) {
	tail_init();
	for (int i=0; i<20; i++) 
	{
		tail_p99(i);
		tail_print();
	}

}
