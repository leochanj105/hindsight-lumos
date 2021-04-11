package main

import (
	// "fmt"
	"sync"

	. "cache"
	. "memory"
	. "queue"
	. "server"
)

func hindsight_init(cap int) {
	SharedPool.Pool = MemInit("/dev/shm/pool", cap*50*4)
	SharedDict.Dict = MemInit("/dev/shm/dict", 3200)
	Complete = QueueInit("/dev/shm/complete_queue", cap)
	Available = QueueInit("/dev/shm/available_queue", cap)
	Triggers = QueueInit("/dev/shm/triggers_queue", cap)

	CacheInit(cap)
	AvailableInit(cap)
	// QueueInitTest()
}

func run() {
	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		RunQueueServer()
		wg.Done()
	}()

	go func() {
		RunTriggerServer()
		wg.Done()
	}()

	wg.Wait()
}

func stat() {
	PrintQueueStat()
}

func main() {
	hindsight_init(100)

	run()
	// stat()

	// queue := QueueInit("/dev/shm/queue_test", 200)

	// for {
	// 	data := QueueGet(queue)
	// 	fmt.Println("get data", data)
	// }

	// QueueTest(queue)
	// mem := MemInit("/dev/shm/pool_test", 1008)

	// fmt.Println(mem[0:8], BytesToInt64(mem[0:8]))
	// fmt.Println(mem[8:12], BytesToInt32(mem[8:12]))
	// for i := 0; i < 250; i++ {
	// 	fmt.Println(mem[12+i*4:16+i*4], BytesToInt32(mem[12+i*4:16+i*4]))
	// }

}
