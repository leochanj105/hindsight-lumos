package server

import (
	"fmt"

	. "cache"
	. "memory"
	. "queue"
)

/*
	QueueServer: keep pooling from queues
	- acquire written buffer from complete
	- once available is empty (or lower than threshold), evict
	- TODO: rate limiting for pooling
*/
func RunQueueServer() {
	var buf int
	var request_id, timestamp int64
	// var evicted map[int]int64
	for true {
		buf = QueueGet(Complete)
		fmt.Println("[server] get buffer", buf, "from complete")

		// buffer management handler
		request_id, timestamp = GetBufMetadata(buf)
		CacheSet(request_id, buf, timestamp)
		CacheManager()

		// if evicted != nil {
		// 	for buffer_id, _ := range evicted {
		// 		fmt.Println("[server] put buffer", buffer_id, "to available")
		// 		QueuePut(Available, buffer_id)
		// 	}
		// }
		PrintQueueStat()
		PrintCacheStat()
	}
}

/*
	TriggerServer: keep pooling triggers, report to Log Collector once detected
*/
func RunTriggerServer() {
	var trigger int
	for true {
		trigger = QueueGet(Triggers)
		fmt.Println("triggers", trigger)
	}
}

/*
	CollectionServer: keep pooling message queues from Log Collector, report upon request
*/

func RunCollectionServer() {

}

/*
	StatServer: local stats
*/
func RunStatServer() {

}
