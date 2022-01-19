package agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
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
	enabled bool

	rate_limit  float64
	buffer_size int
	bucket      *ratelimit.Bucket

	server_port string // The port that the agent listens on for remote triggers

	coordinator_addr string
	coordinator_port string
	localtriggers    chan []memory.Trigger    // Local triggers to be reported to coordinator
	breadcrumbs      chan map[uint64][]string // Breadcrumbs to be reported to coordinator
	remotetriggers   chan []memory.Trigger    // Remote triggers received from coordinator

	collector_addr string
	collector_port string
	data           chan []int // Buffers to be reported to collector
}

func InitReporting(api *memory.GoAgentAPI, rate_limit_mb float64) *Reporting {
	var r Reporting
	r.Init(api, rate_limit_mb)
	return &r
}

func (r *Reporting) Init(api *memory.GoAgentAPI, rate_limit_mb float64) {
	r.api = api
	// r.collector set after run
	r.data = make(chan []int, 4)               // 4 somewhat arbitrary
	r.enabled = true                           // used for testing/dev
	r.rate_limit = rate_limit_mb * 1024 * 1024 // rate limit in bytes/s
	r.buffer_size = r.api.BufferSize()

	if r.rate_limit != 0 {
		r.bucket = ratelimit.NewBucketWithRate(r.rate_limit, int64(r.rate_limit))
	}

	r.server_port = util.Server_port // TODO not in this hacky way

	r.coordinator_addr = util.LC_addr // TODO not in this hacky way
	r.coordinator_port = util.LC_port // TODO not in this hacky way
	r.localtriggers = make(chan []memory.Trigger)
	r.breadcrumbs = make(chan map[uint64][]string)
	r.remotetriggers = make(chan []memory.Trigger)

	r.collector_addr = "" // TODO add separate collection backend addr
	r.collector_port = "" // TODO add separate collection backend port
}

/* Reports trace data to the collector */
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

func (r *Reporting) reportBreadcrumbs(coordinator datapb.CoordinatorClient, accumulated_breadcrumbs []map[uint64][]string) error {
	var request datapb.BreadcrumbsRequest
	request.Src = util.Server_addr + ":" + util.Server_port // TODO anything but this

	for _, breadcrumbs := range accumulated_breadcrumbs {
		for trace_id, addrs := range breadcrumbs {
			var bcs datapb.Breadcrumbs
			bcs.TraceId = trace_id
			bcs.Addrs = addrs
			request.Breadcrumbs = append(request.Breadcrumbs, &bcs)
		}
	}

	if r.enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := coordinator.Breadcrumbs(ctx, &request)

		return err
	}

	return nil
}

func (r *Reporting) reportTriggers(coordinator datapb.CoordinatorClient, triggers []memory.Trigger) error {
	var request datapb.TriggerRequest
	request.Src = util.Server_addr + ":" + util.Server_port // TODO anything but this

	for _, trigger := range triggers {
		var t datapb.Trigger
		t.QueueId = int32(trigger.Queue_id)
		t.BaseTraceId = trigger.Base_trace_id
		t.TraceIds = []uint64{trigger.Trace_id}
		request.Triggers = append(request.Triggers, &t)
	}

	if r.enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := coordinator.LocalTrigger(ctx, &request)

		return err
	}

	return nil
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
		r.remotetriggers <- triggers
	}

	return &datapb.TriggerReply{}, nil
}

func (r *Reporting) BreadcrumbsLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Println("BreadcrumbsLoop goroutine running")
			addr := r.collector_addr + ":" + r.collector_port

			conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithTimeout(100*time.Millisecond))
			if err != nil {
				fmt.Println("Unable to connect to coordinator", addr, err)
				time.Sleep(time.Duration(2) * time.Second)
				continue
			}

			fmt.Println("BreadcrumbsLoop connected to", addr)
			defer conn.Close()

			coordinator := datapb.NewCoordinatorClient(conn)

			err = r.ReportBreadcrumbs(ctx, coordinator)
			if err != nil {
				fmt.Println("Error in BreadcrumbsLoop:", err)
				time.Sleep(time.Duration(2) * time.Second)
				continue
			}
		}
	}
}

func (r *Reporting) ReportBreadcrumbs(ctx context.Context, coordinator datapb.CoordinatorClient) error {
	for {
		// Accumulate a batch of up to 100 breadcrumbs
		var accumulated []map[uint64][]string

	Accumulation:
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				return nil
			case breadcrumbs := <-r.breadcrumbs:
				if len(breadcrumbs) > 0 {
					accumulated = append(accumulated, breadcrumbs)
				}
			default:
				if len(accumulated) > 0 {
					break Accumulation
				}
			}
		}

		err := r.reportBreadcrumbs(coordinator, accumulated)

		if err != nil {
			return err
		}
	}
}

func (r *Reporting) TriggersLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Println("TriggersLoop goroutine running")
			addr := r.collector_addr + ":" + r.collector_port

			conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithTimeout(100*time.Millisecond))
			if err != nil {
				fmt.Println("Unable to connect to coordinator", addr, err)
				time.Sleep(time.Duration(2) * time.Second)
				continue
			}

			fmt.Println("TriggersLoop connected to", addr)
			defer conn.Close()

			coordinator := datapb.NewCoordinatorClient(conn)

			err = r.ReportTriggers(ctx, coordinator)
			if err != nil {
				fmt.Println("Error in TriggersLoop:", err)
				time.Sleep(time.Duration(2) * time.Second)
				continue
			}
		}
	}
}

func (r *Reporting) ReportTriggers(ctx context.Context, coordinator datapb.CoordinatorClient) error {
	for {
		// Accumulate a batch of up to 100 triggers
		var accumulated []memory.Trigger

	Accumulation:
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				return nil
			case triggers := <-r.localtriggers:
				accumulated = append(accumulated, triggers...)
			default:
				if len(accumulated) > 0 {
					break Accumulation
				}
			}
		}

		err := r.reportTriggers(coordinator, accumulated)

		if err != nil {
			return err
		}
	}
}

func (r *Reporting) DataLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Println("DataLoop goroutine running")
			addr := r.collector_addr + ":" + r.collector_port

			conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithTimeout(100*time.Millisecond))
			if err != nil {
				fmt.Println("Unable to connect to coordinator", addr, err)
				time.Sleep(time.Duration(2) * time.Second)
				continue
			}

			fmt.Println("DataLoop connected to", addr)
			defer conn.Close()

			coordinator := datapb.NewCoordinatorClient(conn)

			err = r.ReportData(ctx, coordinator)
			if err != nil {
				fmt.Println("Error in TriggersLoop:", err)
				time.Sleep(time.Duration(2) * time.Second)
				continue
			}
		}
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

	wg := new(sync.WaitGroup)
	wg.Add(3)
	go func() {
		r.BreadcrumbsLoop(ctx)
		wg.Done()
	}()
	go func() {
		r.TriggersLoop(ctx)
		wg.Done()
	}()
	go func() {
		r.DataLoop(ctx)
		wg.Done()
	}()
	wg.Wait()
	s.Stop()
}
