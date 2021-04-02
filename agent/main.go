package main

import (
	"fmt"
	. "queue"
)

func main() {
	queue := QueueInit("/dev/shm/queue_test", 200)

	for {
		data := QueueGet(queue)
		fmt.Println("get data", data)
	}

	// QueueTest(queue)
}
