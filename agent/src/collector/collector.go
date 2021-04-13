package collector

import (
	"fmt"
	"log"
	"net"
	"time"

	. "datapb"
	. "util"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// local pool for all incoming traces (with TTL?)
type trace struct {
	entry      []int32
	trace_data []byte
	addrs      []string
}

// trace might overlap (like different stage of same agent buffer)
var trace_pool map[int64][]trace

// if an address is called 5 times without response, don't try again
var retrieval_pool map[int64]map[string]int
var retrieval_queue MessageQueue
var collection_queue RetrievalQueue

func CollectorInit() {
	trace_pool = make(map[int64][]trace)
	retrieval_pool = make(map[int64]map[string]int)
	retrieval_queue.Req = make(map[int64]int)
	collection_queue.Req = make(map[int64]map[string]int)
}

type collectorServer struct {
}

func newCollector() *collectorServer {
	s := &collectorServer{}
	return s
}

/*
	collectorServer.Report: add agent reported traces to TracePool
*/
func (*collectorServer) Report(ctx context.Context, in *Trace) (*CallRet, error) {
	request_id := in.RequestId
	fmt.Println("[Log Collector] receiving", request_id)
	if _, ok := trace_pool[request_id]; !ok {
		var temp []trace
		trace_pool[request_id] = temp
	}
	trace_pool[request_id] = append(trace_pool[request_id], trace{entry: in.Entry, trace_data: in.Trace, addrs: in.Addrs})

	// notify collector
	retrieval_queue.Mutex.Lock()
	retrieval_queue.Req[request_id] = 1
	retrieval_queue.Mutex.Unlock()

	return &CallRet{Callret: true}, nil
}

/*
	LCServer: listen to agents
*/
func RunLCResponseServer() {
	for true {
		lis, err := net.Listen("tcp", ":"+LC_port)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
			fmt.Println("failed to listen:", err)
		}
		s := grpc.NewServer()
		RegisterCollectorServer(s, newCollector())
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
			fmt.Println("failed to serve:", err)
		}
	}

}

/*
	RetrievalHandler: pool incoming traces, find out nodes to retrieve data
*/
func RunRetrievalHandler() {
	// poll retrieval_queue, check neighboring addresses, push to collection_queue
	for true {
		pending := make(map[int64]map[string]int)
		retrieval_queue.Mutex.Lock()
		for request_id, _ := range retrieval_queue.Req {
			pending[request_id] = make(map[string]int)
		}
		retrieval_queue.Req = make(map[int64]int)
		retrieval_queue.Mutex.Unlock()

		for request_id, _ := range pending {
			if _, ok := retrieval_pool[request_id]; !ok {
				retrieval_pool[request_id] = make(map[string]int)
				// retrieval_pool[request_id]["finish"] = 0
				retrieval_pool[request_id]["offset"] = 0
			}
			offset := retrieval_pool[request_id]["offset"]

			for i, trace := range trace_pool[request_id] {
				if i < offset {
					continue
				}
				for _, addr := range trace.addrs {
					if counter, ok := retrieval_pool[request_id][addr]; !ok {
						retrieval_pool[request_id][addr] = 0
						pending[request_id][addr] = 1
					} else {
						if counter < 5 {
							pending[request_id][addr] = 1
						}
					}
					retrieval_pool[request_id][addr] += 1
				}
				retrieval_pool[request_id]["offset"] += 1
			}
		}

		collection_queue.Mutex.Lock()
		for request_id, _ := range pending {
			if len(pending[request_id]) == 0 {
				continue
			}
			for addr, _ := range pending[request_id] {
				if _, ok := collection_queue.Req[request_id]; !ok {
					collection_queue.Req[request_id] = make(map[string]int)
				}
				collection_queue.Req[request_id][addr] = 1
			}
		}
		collection_queue.Mutex.Unlock()
	}
}

/*
	Collector: pool collection_queue, send retrieval request to agents
*/
func RunCollector() {
	var time_counter int = 0
	// poll from collection queue, send request
	for true {
		time_counter += 1
		if time_counter == 10000000 {
			fmt.Println("[log collector] have collected", len(retrieval_pool), "requests")
			time_counter = 0
		}
		pending := make(map[string]map[int64]int)
		collection_queue.Mutex.Lock()
		for request_id, _ := range collection_queue.Req {
			for addr, _ := range collection_queue.Req[request_id] {
				if _, ok := pending[addr]; !ok {
					pending[addr] = make(map[int64]int)
				}
				pending[addr][request_id] = 1
			}
		}
		collection_queue.Req = make(map[int64]map[string]int)
		collection_queue.Mutex.Unlock()

		for addr, _ := range pending {
			var request_ids []int64
			for request_id, _ := range pending[addr] {
				request_ids = append(request_ids, request_id)
			}

			conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(1000000000*time.Nanosecond))
			if err != nil {
				fmt.Println("dial", addr, err)
				// return
				continue
			}
			defer conn.Close()
			c := NewAgentClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Nanosecond)
			defer cancel()

			_, err = c.Request(ctx, &RequestID{
				Rid: request_ids})

			if err != nil {
				fmt.Println("request", err)
				// return
				continue
			}
		}
	}
}
