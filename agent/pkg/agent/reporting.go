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
	api       *memory.GoAgentAPI // API to the shared memory
	collector datapb.CollectorClient
	queue     chan []int
	enabled   bool

	rate_limit  float64
	buffer_size int
	bucket      *ratelimit.Bucket
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

		ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Nanosecond)
		defer cancel()

		// TODO: report differently; not as RPC, and buffers only
		// TODO: configurable whether to actually report or not.
		r.collector.Report(ctx, &datapb.Trace{
			RequestId: int64(0),
			Entry:     entry,
			Trace:     trace_data,
			Addrs:     addrs})
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

/* gRPC requests from Log collector */
func (r *Reporting) Request(ctx context.Context, in *datapb.RequestID) (*datapb.CallRet, error) {
	request_ids := in.Rid

	triggers := make([]memory.Trigger, 0, len(request_ids))

	for _, request_id := range request_ids {
		// TODO  memory.Trigger should categorize as locally or remote
		triggers = append(triggers, memory.Trigger{2, uint64(request_id), uint64(request_id)})
	}

	if len(triggers) > 0 {
		r.api.Triggers <- triggers
	}

	return &datapb.CallRet{Callret: true}, nil
}

func (r *Reporting) Run(ctx context.Context) {
	fmt.Println("Reporting goroutine running")
	conn, err := grpc.Dial(util.LC_addr+":"+util.LC_port,
		grpc.WithInsecure(),
		grpc.WithTimeout(100000000*time.Nanosecond))
	if err != nil {
		fmt.Println("dial", util.LC_addr+":"+util.LC_port, err)
		return
	}
	fmt.Println("Reporting connected to", util.LC_addr+":"+util.LC_port)
	defer conn.Close()

	lis, err := net.Listen("tcp", ":"+util.Server_port)
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

	r.collector = datapb.NewCollectorClient(conn)
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
