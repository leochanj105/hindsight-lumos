package server

import (
	"fmt"
	"log"
	"net"
	"time"

	. "cache"
	. "datapb"
	. "memory"
	. "queue"
	. "util"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// TODO: potential data lost here: request_id might be registered, but data might be 1) not coming yet, 2) right there, and 3) flushed when sending to LC.
// Possible solution: copy data to a secured pool once detected
var report_queue MessageQueue

func ServerInit() {
	report_queue.Req = make(map[int64]int)
}

/*
	QueueServer: pooling from complete, evict buffers (LRU) and push back to available
	- TODO: rate limiting for pooling
*/
func RunQueueServer() {
	var buf int
	var request_id, timestamp int64
	// var evicted map[int]int64
	for true {
		buf = QueueGet(Complete)
		if buf == -1 {
			continue
		}
		// if DEBUG == 1 {
		// 	fmt.Println("[server] get buffer", buf, "from complete")
		// }

		// buffer management handler
		request_id, timestamp = GetBufMetadata(buf)
		CacheSet(request_id, buf, timestamp)
		CacheManager()

		// PrintQueueStat()
		// PrintCacheStat()
	}
}

/*
	TriggerServer: keep pooling triggers, add request to report_queue
*/
func RunTriggerServer() {
	var request_id int64
	for true {
		temp := QueueGet(Triggers)
		if temp != -1 {
			request_id = (int64)(temp << 32)
			var temp2 int
			for true {
				temp2 = QueueGet(Triggers)
				if temp2 == -1 {
					continue
				}
				request_id += int64(temp2)
				break
			}
			report_queue.Mutex.Lock()
			// if same request is added again, reset counter to 0
			report_queue.Req[request_id] = 0
			report_queue.Mutex.Unlock()
			if DEBUG == 1 {
				fmt.Println("[trigger server] find trigger of request", request_id)
			}
		}
	}
}

type agentServer struct {
}

func newAgent() *agentServer {
	s := &agentServer{}
	return s
}

/*
	agentServer.Request: add LC requests to report_queue
*/
func (*agentServer) Request(ctx context.Context, in *RequestID) (*CallRet, error) {
	request_ids := in.Rid
	report_queue.Mutex.Lock()
	for _, request_id := range request_ids {
		report_queue.Req[request_id] = 0
		fmt.Println("receiving", request_id)
	}
	report_queue.Mutex.Unlock()

	return &CallRet{Callret: true}, nil
}

/*
	ResponseServer: listen to log collector
*/
func RunResponseServer() {
	lis, err := net.Listen("tcp", ":"+Server_port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	RegisterAgentServer(s, newAgent())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

/*
	Agent: pool report_queue, send to log collector
*/
func RunAgent() {
	pending := make(map[int64]int)
	for true {
		reported := make(map[int64]bool)
		report_queue.Mutex.Lock()
		for request_id, counter := range report_queue.Req {
			if _, ok := pending[request_id]; ok {
				if counter >= 10 {
					reported[request_id] = true
				} else {
					pending[request_id] += 1
				}
			} else {
				pending[request_id] = 0
			}

		}
		report_queue.Req = make(map[int64]int)
		report_queue.Mutex.Unlock()

		for request_id, _ := range pending {
			buffer_ids := CacheGetBuffers(request_id)
			if buffer_ids == nil {
				continue
			}
			var entry []int32
			var trace_data []byte
			var addrs []string
			addrs = append(addrs, Server_addr+":"+Server_port)
			for buf, _ := range buffer_ids {
				offset := buf * Buf_length * 4
				entry = append(entry, int32(buf))
				trace_data = append(trace_data, SharedPool.Pool[offset:offset+Buf_length*4]...)
				breadcrumbs := GetBreadcrumbs(buf)
				for _, breadcrumb := range breadcrumbs {
					addrs = append(addrs, string(SharedDict.Dict[breadcrumb*32:(breadcrumb+1)*32]))
				}
			}

			conn, err := grpc.Dial(LC_addr+":"+LC_port, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(100000000*time.Nanosecond))
			if err != nil {
				fmt.Println("dial", LC_addr+":"+LC_port, err)
				return
			}
			defer conn.Close()
			c := NewCollectorClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), 100000000*time.Nanosecond)
			defer cancel()

			_, err = c.Report(ctx, &Trace{
				RequestId: request_id,
				Entry:     entry,
				Trace:     trace_data,
				Addrs:     addrs})

			if err != nil {
				fmt.Println("report", err)
				return
			}

			reported[request_id] = true
		}

		for request_id, _ := range reported {
			delete(pending, request_id)
		}
	}
}
