package collector_new

import (
	"context"
	"sync"
)

type trace struct {
	request_id int64
	entry      []int32
	trace_data []byte
	addrs      []string
}

type breadcrumbs struct {
	request_id int64
	addrs      []string
}

type Collector struct {
	// datapb.UnimplementedCollectorServer

	// trace_pool keeps all coming trace data with {request_id : {addr: [traces]}}
	trace_pool       map[int64]map[string][]trace
	trace_pool_mutex sync.RWMutex

	// retrieval_pool keeps all requested agents given request_id. Each agent will only be requested once
	retrieval_pool       map[int64]map[string]bool
	retrieval_pool_mutex sync.RWMutex

	coming_report   chan breadcrumbs
	pending_request chan breadcrumbs
}

func InitLC() *Collector {
	var collector Collector
	new_trace_pool := make(map[int64]map[string][]trace)
	collector.trace_pool = new_trace_pool

	new_retrieval_pool := make(map[int64]map[string]bool)
	collector.retrieval_pool = new_retrieval_pool

	collector.coming_report = make(chan breadcrumbs, 1000)
	collector.pending_request = make(chan breadcrumbs, 1000)
	return &collector
}

func (lc *Collector) Run(ctx context.Context) {
	// wg := new(sync.WaitGroup)
	// wg.Add(2)
	// go func() {
	// 	lc.RunLCServer()
	// 	wg.Done()
	// }()
	// go func() {
	// 	lc.RunCollector(ctx)
	// 	wg.Done()
	// }()
	// wg.Wait()
}

// func (lc *Collector) RunLCServer() {
// 	for true {
// 		lis, err := net.Listen("tcp", ":"+util.LC_port)
// 		if err != nil {
// 			log.Fatalf("failed to listen: %v", err)
// 		}
// 		s := grpc.NewServer()
// 		datapb.RegisterCollectorServer(s, lc)
// 		if err := s.Serve(lis); err != nil {
// 			log.Fatalf("failed to serve: %v", err)
// 		}
// 	}
// }

// /* gRPC report from agent */
// func (lc *Collector) Report(ctx context.Context, in *datapb.Trace) (*datapb.CallRet, error) {
// 	// add data to trace_pool and retrieval channel
// 	request_id := in.RequestId
// 	addr := in.Addrs[0]
// 	var new_trace trace = trace{request_id: in.RequestId, entry: in.Entry, trace_data: in.Trace, addrs: in.Addrs}
// 	lc.trace_pool_mutex.Lock()
// 	if _, ok := lc.trace_pool[request_id]; !ok {
// 		lc.trace_pool[request_id] = make(map[string][]trace)
// 	}
// 	lc.trace_pool[request_id][addr] = append(lc.trace_pool[request_id][addr], new_trace)
// 	lc.trace_pool_mutex.Unlock()
// 	fmt.Printf("[Log Collector] Received %d from %s length %d %d\n", request_id, addr, len(in.Trace), util.GetTime())

// 	lc.coming_report <- breadcrumbs{request_id: in.RequestId, addrs: in.Addrs}

// 	return &datapb.CallRet{Callret: true}, nil
// }

// func (lc *Collector) RunCollector(ctx context.Context) {
// 	// receive from retrieval channel, check with retrieval_pool, add to a queue for request
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case new_report := <-lc.coming_report:
// 			{
// 				fmt.Printf("Received request %d\n", new_report.request_id)
// 				lc.retrieval_pool_mutex.Lock()
// 				if _, ok := lc.retrieval_pool[new_report.request_id]; !ok {
// 					lc.retrieval_pool[new_report.request_id] = make(map[string]bool)
// 					for _, addr := range new_report.addrs {
// 						lc.retrieval_pool[new_report.request_id][addr] = true
// 					}
// 					lc.retrieval_pool_mutex.Unlock()
// 					lc.pending_request <- new_report
// 				} else {
// 					var pending_addrs []string
// 					for _, addr := range new_report.addrs {
// 						if _, okk := lc.retrieval_pool[new_report.request_id][addr]; !okk {
// 							lc.retrieval_pool[new_report.request_id][addr] = true
// 							pending_addrs = append(pending_addrs, addr)
// 						}
// 					}
// 					lc.retrieval_pool_mutex.Unlock()
// 					lc.pending_request <- breadcrumbs{request_id: new_report.request_id, addrs: pending_addrs}
// 				}
// 			}
// 		case new_requests := <-lc.pending_request:
// 			{
// 				fmt.Printf("Sending request %d\n", new_requests.request_id)
// 				request_id := new_requests.request_id
// 				for _, addr := range new_requests.addrs {
// 					conn, err := grpc.Dial(addr,
// 						grpc.WithInsecure(),
// 						grpc.WithTimeout(100000000*time.Nanosecond))
// 					if err != nil {
// 						fmt.Println("dial", addr, err)
// 						conn.Close()
// 						return
// 					}
// 					// fmt.Println("Collector connected to", addr)
// 					// defer conn.Close()

// 					c := datapb.NewAgentClient(conn)

// 					ctx_, cancel := context.WithTimeout(context.Background(), 1000000000*time.Nanosecond)
// 					defer cancel()

// 					request_ids := []int64{request_id}
// 					_, err = c.Request(ctx_, &datapb.RequestID{
// 						Rid: request_ids})

// 					if err != nil {
// 						fmt.Println("request", err)
// 						// return
// 					}
// 					conn.Close()
// 				}

// 			}
// 		}

// 	}
// }
