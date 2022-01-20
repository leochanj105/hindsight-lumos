package coordinator

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/geraldleizhang/hindsight/agent/pkg/datapb"
	"google.golang.org/grpc"
)

type CoordinatorServer struct {
	datapb.UnimplementedCoordinatorServer

	c      Coordinator       // Manages coordination data
	agents map[string]*Agent // connections to agents

	listen_port string // Port to listen for connections from agents

	incoming_triggers    chan *datapb.TriggerRequest
	incoming_breadcrumbs chan *datapb.BreadcrumbsRequest
}

type Agent struct {
	addr              string
	outgoing_triggers chan []Trigger
}

func (s *CoordinatorServer) Init(port string) {
	s.c.Init()
	s.agents = make(map[string]*Agent)
	s.listen_port = port
	s.incoming_triggers = make(chan *datapb.TriggerRequest, 1000)
	s.incoming_breadcrumbs = make(chan *datapb.BreadcrumbsRequest, 1000)
}

func (s *CoordinatorServer) Run(ctx context.Context) {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		s.runServer()
		wg.Done()
	}()
	go func() {
		s.runCoordinator(ctx)
		wg.Done()
	}()
	wg.Wait()
}

/* Run the RPC server that receives triggers and breadcrumbs */
func (cs *CoordinatorServer) runServer() {
	for true {
		lis, err := net.Listen("tcp", ":"+cs.listen_port)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		fmt.Println("Listening for agent connections on port", cs.listen_port)
		grpcserver := grpc.NewServer()
		datapb.RegisterCoordinatorServer(grpcserver, cs)
		if err := grpcserver.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}
}

func (cs *CoordinatorServer) GetAgent(addr string) *Agent {
	if agent, ok := cs.agents[addr]; ok {
		return agent
	} else {
		var agent Agent
		agent.Init(addr)
		cs.agents[addr] = &agent
		return &agent
	}
}

func (cs *CoordinatorServer) processTriggersRequest(req *datapb.TriggerRequest) {
	triggers_to_forward := make(map[string][]Trigger)
	for _, t := range req.Triggers {
		// Store the received trigger
		var trigger Trigger
		trigger.id.queue_id = int(t.QueueId)
		trigger.id.base_trace_id = t.BaseTraceId
		trigger.trace_ids = t.TraceIds
		forwarding_addrs := cs.c.AddTrigger(req.Src, trigger)

		// Forward the trigger to any addresses specified
		for _, addr := range forwarding_addrs {
			triggers_to_forward[addr] = append(triggers_to_forward[addr], trigger)
		}
	}

	// Do the forwarding
	for addr, triggers := range triggers_to_forward {
		cs.GetAgent(addr).SendTriggers(triggers)
	}
}

func (cs *CoordinatorServer) processBreadcrumbRequest(req *datapb.BreadcrumbsRequest) {
	// Breadcrumbs are received inverted; reverse this
	var inverted map[uint64][]string
	for _, b := range req.Breadcrumbs {
		for _, trace_id := range b.TraceIds {
			inverted[trace_id] = append(inverted[trace_id], b.Addr)
		}
	}

	// Now process them
	triggers_to_forward := make(map[string][]Trigger)
	for trace_id, addrs := range inverted {
		// Store the received breadcrumbs
		to_forward := cs.c.AddBreadcrumb(req.Src, trace_id, addrs)

		// Forward any necessary triggers
		for addr, triggers := range to_forward {
			if len(triggers) > 0 {
				triggers_to_forward[addr] = append(triggers_to_forward[addr], triggers...)
			}
		}
	}

	// Do the forwarding
	for addr, triggers := range triggers_to_forward {
		cs.GetAgent(addr).SendTriggers(triggers)
	}
}

/* The "main" thread that receives incoming stuff and sends outgoing stuff */
func (cs *CoordinatorServer) runCoordinator(ctx context.Context) {
	fmt.Println("CoordinatorServer main goroutine running")
	for {
		select {
		case <-ctx.Done():
			fmt.Println("CoordinatorServer main goroutine exiting")
			return
		case req := <-cs.incoming_triggers:
			/* Received some triggers from an agent over RPC*/
			cs.processTriggersRequest(req)
		case req := <-cs.incoming_breadcrumbs:
			/* Received some breadcrumbs from an agent over RPC */
			cs.processBreadcrumbRequest(req)
		}
	}
}

/* An agent has sent us a trigger */
func (s *CoordinatorServer) LocalTrigger(ctx context.Context, in *datapb.TriggerRequest) (*datapb.TriggerReply, error) {
	fmt.Println("Received a local trigger!", in.Triggers)
	select {
	case s.incoming_triggers <- in:
		break
	default:
		fmt.Println("LocalTrigger incoming_triggers bottlenecked!")
	}
	return &datapb.TriggerReply{}, nil
}

/* An agent has sent us breadcrumbs */
func (s *CoordinatorServer) Breadcrumbs(ctx context.Context, in *datapb.BreadcrumbsRequest) (*datapb.BreadcrumbsReply, error) {
	fmt.Println("Received breadcrumbs!", in.Breadcrumbs)
	select {
	case s.incoming_breadcrumbs <- in:
		break
	default:
		fmt.Println("LocalTrigger incoming_breadcrumbs bottlenecked!")
	}
	return &datapb.BreadcrumbsReply{}, nil
}

func (a *Agent) Init(addr string) {
	a.addr = addr
}

func (a *Agent) Run() {

}

func (a *Agent) SendTriggers(triggers []Trigger) {
	fmt.Println("Forwarding triggers!", a.addr, triggers)
}
