package main

import (
	queue "queue"
)

func main() {
	queue.QueueInit("/dev/shm/queue_test", 200)
}
