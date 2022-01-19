package agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/geraldleizhang/hindsight/agent/pkg/datapb"
	"github.com/geraldleizhang/hindsight/agent/pkg/memory"
	"github.com/geraldleizhang/hindsight/agent/pkg/util"
	"google.golang.org/grpc"

	"github.com/juju/ratelimit"
)

type Reporting struct {
	datapb.UnimplementedAgentServer

	api     *memory.GoAgentAPI // API to the shared memory
	queue   chan []int
	enabled bool

	rate_limit  float64
	buffer_size int
	bucket      *ratelimit.Bucket

	server_port string // The port that the agent listens on for remote triggers

	coordinator      datapb.CoordinatorClient
	coordinator_addr string
	coordinator_port string

	collector_addr string
	collector_port string
}

func InitReporting(api *memory.GoAgentAPI, rate_limit_mb float64) *Reporting {
	var r Reporting
	r.Init(api, rate_limit_mb)
	return &r
}

func (r *Reporting) Init(api *memory.GoAgentAPI, rate_limit_mb float64) {
	r.api = api
	// r.collector set after run
	r.queue = make(chan []int, 4)              // 4 somewhat arbitrary
	r.enabled = true                           // used for testing/dev
	r.rate_limit = rate_limit_mb * 1024 * 1024 // rate limit in bytes/s
	r.buffer_size = r.api.BufferSize()

	if r.rate_limit != 0 {
		r.bucket = ratelimit.NewBucketWithRate(r.rate_limit, int64(r.rate_limit))
	}

	r.server_port = util.Server_port  // TODO not in this hacky way
	r.coordinator_addr = util.LC_addr // TODO not in this hacky way
	r.coordinator_port = util.LC_port // TODO not in this hacky way
	r.collector_addr = ""             // TODO add separate collection backend addr
	r.collector_port = ""             // TODO add separate collection backend port
}

/* Reports trace data to the collector */
func (r *Reporting) report(buffers []int) {
	// fmt.Printf("Reporting trace %d with %d buffers, breadcrumbs: ", trace_id, len(trace.Buffers))
	// for i, addr := range trace.Breadcrumbs {
	// 	fmt.Printf("(%d: %s) ", i, addr)
	// }
	// fmt.Printf("\n")

	if r.bucket != nil {
		r.bucket.Wait(int64(len(buffers) * r.buffer_size))
	}

	if r.enabled {
		var entry []int32
		var trace_data []byte
		var addrs []string
		addrs = append(addrs, util.Server_addr+":"+util.Server_port)

		for _, buffer_id := range buffers {
			entry = append(entry, int32(buffer_id))
			data := r.api.GetBuffer(buffer_id)
			trace_data = append(trace_data, data...)
		}

		// ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Nanosecond)
		// defer cancel()

		// TODO: report differently; not as RPC, and buffers only
		// TODO: configurable whether to actually report or not.
		// r.collector.Report(ctx, &datapb.Trace{
		// 	RequestId: int64(0),
		// 	Entry:     entry,
		// 	Trace:     trace_data,
		// 	Addrs:     addrs})
	}

	// if err != nil {
	// 	fmt.Println("report", err)
	// 	// return
	// }

	// Return the buffers
	if len(buffers) > 0 {
		r.api.Available <- buffers
	}
}

/* remote trigger from coordinator over RPC */
func (r *Reporting) RemoteTrigger(ctx context.Context, in *datapb.TriggerRequest) (*datapb.TriggerReply, error) {

	var triggers []memory.Trigger
	for _, trigger := range in.Triggers {
		// TODO  memory.Trigger should categorize as locally or remote
		// should combine all triggers into one
		// should use a separate channel to local triggers
		for _, traceid := range trigger.GetTraceIds() {
			var mt memory.Trigger
			mt.Queue_id = int(trigger.QueueId)
			mt.Base_trace_id = trigger.BaseTraceId
			mt.Trace_id = traceid
			triggers = append(triggers, mt)
		}
	}

	if len(triggers) > 0 {
		r.api.Triggers <- triggers
	}

	return &datapb.TriggerReply{}, nil
}

func (r *Reporting) Run(ctx context.Context) {
	fmt.Println("Reporting goroutine running")
	addr := r.collector_addr + ":" + r.collector_port

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithTimeout(100*time.Millisecond))
	if err != nil {
		fmt.Println("dial", addr, err)
		return
	}
	fmt.Println("Reporting connected to", addr)
	defer conn.Close()

	lis, err := net.Listen("tcp", ":"+r.server_port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	datapb.RegisterAgentServer(s, r)

	// Run the GRPC server
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("GRPC Server: %v", err)
		}
	}()

	// r.collector = datapb.NewCollectorClient(conn)
	for {
		select {
		case <-ctx.Done():
			s.Stop()
			return
		case buffers := <-r.queue:
			r.report(buffers)
		}
	}
}
