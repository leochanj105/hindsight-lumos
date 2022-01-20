package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/geraldleizhang/hindsight/agent/pkg/datapb"
	"github.com/geraldleizhang/hindsight/agent/pkg/memory"
	"github.com/geraldleizhang/hindsight/agent/pkg/util"
	"google.golang.org/grpc"

	"github.com/juju/ratelimit"
)

type Reporting struct {
	api     *memory.GoAgentAPI // API to the shared memory for returning data buffers
	enabled bool

	rate_limit  float64
	buffer_size int
	bucket      *ratelimit.Bucket

	remote_addr string     // Address of the trace data backend (not the coordinator)
	data        chan []int // Buffers to be reported to collector
}

func InitReporting(api *memory.GoAgentAPI, rate_limit_mb float64, enabled bool) *Reporting {
	var r Reporting
	r.Init(api, rate_limit_mb, enabled)
	return &r
}

func (r *Reporting) Init(api *memory.GoAgentAPI, rate_limit_mb float64, enabled bool) {
	r.api = api
	r.data = make(chan []int, 4)               // 4 somewhat arbitrary
	r.enabled = enabled                        // used for testing/dev
	r.rate_limit = rate_limit_mb * 1024 * 1024 // rate limit in bytes/s
	r.buffer_size = r.api.BufferSize()

	if r.rate_limit != 0 {
		r.bucket = ratelimit.NewBucketWithRate(r.rate_limit, int64(r.rate_limit))
	}

	r.remote_addr = util.Reporting_addr + ":" + util.Reporting_port
}

/* Reports trace data to the collector TODO grpc? */
func (r *Reporting) reportData(buffers []int) error {
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

	return nil
}

/* TODO: not sure we'll actually use grpc for reporting */
func (r *Reporting) DataLoop(ctx context.Context) {
	fmt.Println("DataLoop connecting to", r.remote_addr)
	firsttime := true
	for {
		select {
		case <-ctx.Done():
			return
		default:
			break
		}

		conn, err := grpc.Dial(r.remote_addr, grpc.WithInsecure(), grpc.WithTimeout(100*time.Millisecond))
		if err != nil {
			if firsttime {
				fmt.Println("Unable to connect to reporting backend, retrying every 2 seconds", r.remote_addr, err)
				firsttime = false
			}
			time.Sleep(time.Duration(2) * time.Second)
			continue
		}
		defer conn.Close()

		coordinator := datapb.NewCoordinatorClient(conn)

		err = r.ReportData(ctx, coordinator)
		if err != nil {
			if firsttime {
				fmt.Println("Error in DataLoop:", err, " -- will retry every 2 seconds")
				firsttime = false
			}
			time.Sleep(time.Duration(2) * time.Second)
			continue
		}

		firsttime = true
	}
}

func (r *Reporting) ReportData(ctx context.Context, coordinator datapb.CoordinatorClient) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case buffers := <-r.data:
			err := r.reportData(buffers)
			if err != nil {
				return err
			}
		}
	}
}

func (r *Reporting) Run(ctx context.Context) {
	fmt.Println("Reporting goroutine running")

	r.DataLoop(ctx)
}
