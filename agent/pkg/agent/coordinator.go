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
	"google.golang.org/grpc"
)

type Coordinator struct {
	datapb.UnimplementedAgentServer

	enabled bool

	local_addr  string // The address that the coordinator uses to contact us
	local_port  string
	remote_addr string // Address of the coordinator

	localtriggers  chan []memory.Trigger    // Local triggers to be reported to coordinator
	breadcrumbs    chan map[uint64][]string // Breadcrumbs to be reported to coordinator
	remotetriggers chan []memory.Trigger    // Remote triggers received from coordinator
}

func InitCoordinator(enabled bool, local_hostname string, local_port string, remote_addr string) *Coordinator {
	var r Coordinator
	r.Init(enabled, local_hostname, local_port, remote_addr)
	return &r
}

func (r *Coordinator) Init(enabled bool, local_hostname string, local_port string, remote_addr string) {
	r.enabled = enabled // used for testing/dev

	r.local_port = local_port
	r.local_addr = local_hostname + ":" + local_port
	r.remote_addr = remote_addr

	r.localtriggers = make(chan []memory.Trigger, 500)
	r.breadcrumbs = make(chan map[uint64][]string, 500)
	r.remotetriggers = make(chan []memory.Trigger, 500)
}

/* Send a batch of breadcrumbs to the coordinator */
func (r *Coordinator) sendBreadcrumbs(rpcclient datapb.CoordinatorClient, accumulated_breadcrumbs []map[uint64][]string) error {
	// For the RPC call we invert the map to avoid duplicating strings
	inverted := make(map[string][]uint64)
	for _, breadcrumbs := range accumulated_breadcrumbs {
		for trace_id, addrs := range breadcrumbs {
			for _, addr := range addrs {
				inverted[addr] = append(inverted[addr], trace_id)
			}
		}
	}

	// Construct RPC request object
	var request datapb.BreadcrumbsRequest
	request.Src = r.local_addr
	for addr, trace_ids := range inverted {
		var bcs datapb.Breadcrumbs
		bcs.Addr = addr
		bcs.TraceIds = trace_ids
		request.Breadcrumbs = append(request.Breadcrumbs, &bcs)
	}

	if r.enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := rpcclient.Breadcrumbs(ctx, &request)

		return err
	}
	return nil
}

/* Send a batch of local triggers to the coordinator */
func (r *Coordinator) sendTriggers(rpcclient datapb.CoordinatorClient, triggers []memory.Trigger) error {
	var request datapb.TriggerRequest
	request.Src = r.local_addr

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

		_, err := rpcclient.LocalTrigger(ctx, &request)

		return err
	}

	return nil
}

/* Received a batch of remote triggers from the coordinator */
func (r *Coordinator) RemoteTrigger(ctx context.Context, in *datapb.TriggerRequest) (*datapb.TriggerReply, error) {

	var triggers []memory.Trigger
	for _, trigger := range in.Triggers {
		for _, traceid := range trigger.GetTraceIds() {
			// TODO: update memory.Trigger with list of trace ids
			var mt memory.Trigger
			mt.Queue_id = int(trigger.QueueId)
			mt.Base_trace_id = trigger.BaseTraceId
			mt.Trace_id = traceid
			triggers = append(triggers, mt)
		}
	}

	if len(triggers) > 0 {
		select {
		case r.remotetriggers <- triggers:
			break
		default:
			// Agent is bottlenecked, drop remote triggers
			// TODO: counters here
		}
	}

	return &datapb.TriggerReply{}, nil
}

/* After we are connected to the coordinator, this loops over the outgoing breadcrumbs, reporting them in batches */
func (r *Coordinator) BreadcrumbsLoop(ctx context.Context) {
	fmt.Println("BreadcrumbsLoop connecting to", r.remote_addr)
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
				fmt.Println("Unable to connect to coordinator, retrying every 2 seconds", r.remote_addr, err)
				firsttime = false
			}
			time.Sleep(time.Duration(2) * time.Second)
			continue
		}
		defer conn.Close()

		rpcclient := datapb.NewCoordinatorClient(conn)

		err = r.ReportBreadcrumbs(ctx, rpcclient)
		if err != nil {
			if firsttime {
				fmt.Println("Error in BreadcrumbsLoop:", err, " -- will retry every 2 seconds")
				firsttime = false
			}
			time.Sleep(time.Duration(2) * time.Second)
			continue
		}

		firsttime = true
	}
}

func (r *Coordinator) ReportBreadcrumbs(ctx context.Context, rpcclient datapb.CoordinatorClient) error {
	for {
		// Accumulate a batch of up to 100 breadcrumbs
		var accumulated []map[uint64][]string

		// Block waiting for some breadcrumbs
		for len(accumulated) == 0 {
			select {
			case breadcrumbs := <-r.breadcrumbs:
				if len(breadcrumbs) > 0 {
					accumulated = append(accumulated, breadcrumbs)
				}
			}
		}

		// Now try to batch as many additional breadcrumbs as possible (without blocking)
	Accumulation:
		for len(accumulated) < 100 {
			select {
			case <-ctx.Done():
				return nil
			case breadcrumbs := <-r.breadcrumbs:
				if len(breadcrumbs) > 0 {
					accumulated = append(accumulated, breadcrumbs)
				}
			default:
				break Accumulation
			}
		}

		// Send them
		err := r.sendBreadcrumbs(rpcclient, accumulated)

		if err != nil {
			return err
		}
	}
}

func (r *Coordinator) TriggersLoop(ctx context.Context) {
	fmt.Println("TriggersLoop connecting to", r.remote_addr)
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
				fmt.Println("Unable to connect to coordinator, retrying every 2 seconds", r.remote_addr, err)
				firsttime = false
			}
			time.Sleep(time.Duration(2) * time.Second)
			continue
		}
		defer conn.Close()

		rpcclient := datapb.NewCoordinatorClient(conn)

		err = r.ReportTriggers(ctx, rpcclient)
		if err != nil {
			if firsttime {
				fmt.Println("Error in TriggersLoop:", err, " -- will retry every 2 seconds")
				firsttime = false
			}
			time.Sleep(time.Duration(2) * time.Second)
			continue
		}

		firsttime = true
	}
}

func (r *Coordinator) ReportTriggers(ctx context.Context, rpcclient datapb.CoordinatorClient) error {
	for {
		// Accumulate a batch of up to 100 triggers
		var accumulated []memory.Trigger

		// Block waiting for some triggers
		for len(accumulated) == 0 {
			select {
			case triggers := <-r.localtriggers:
				accumulated = append(accumulated, triggers...)
			}
		}

		// Now try to batch as many additional triggers as possible (without blocking)
	Accumulation:
		for len(accumulated) < 100 {
			select {
			case <-ctx.Done():
				return nil
			case triggers := <-r.localtriggers:
				accumulated = append(accumulated, triggers...)
			default:
				break Accumulation
			}
		}

		// Send them
		err := r.sendTriggers(rpcclient, accumulated)

		if err != nil {
			return err
		}
	}
}

func (r *Coordinator) Run(ctx context.Context) {
	fmt.Println("Coordinator goroutine running - coordinator at: ", r.remote_addr)

	lis, err := net.Listen("tcp", ":"+r.local_port)
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
	wg.Add(2)
	go func() {
		r.BreadcrumbsLoop(ctx)
		wg.Done()
	}()
	go func() {
		r.TriggersLoop(ctx)
		wg.Done()
	}()
	wg.Wait()
	s.Stop()
}
