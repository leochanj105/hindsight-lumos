package main

import (
	"fmt"
	. "memory"
	. "queue"
)

func main() {
	// queue := QueueInit("/dev/shm/queue_test", 200)

	// for {
	// 	data := QueueGet(queue)
	// 	fmt.Println("get data", data)
	// }

	// QueueTest(queue)
	mem := MemInit("/dev/shm/pool_test", 1008)

	fmt.Println(mem[0:8], BytesToInt64(mem[0:8]))
	fmt.Println(mem[8:12], BytesToInt32(mem[8:12]))
	for i := 0; i < 250; i++ {
		fmt.Println(mem[12+i*4:16+i*4], BytesToInt32(mem[12+i*4:16+i*4]))
	}

}
