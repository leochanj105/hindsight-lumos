package coordinator

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/geraldleizhang/hindsight/agent/pkg/datapb"
	"google.golang.org/grpc"
)

type IncomingBreadcrumbs struct {
	req *datapb.BreadcrumbsRequest
	ret chan error
}

type IncomingTriggers struct {
	req *datapb.TriggerRequest
	ret chan error
}

type CoordinatorServer struct {
	datapb.UnimplementedCoordinatorServer

	ctx     context.Context   // For shutdown
	timeout time.Duration     // Time before expiring triggers / traces
	c       Coordinator       // Manages coordination data
	agents  map[string]*Agent // connections to agents

	listen_port string // Port to listen for connections from agents

	incoming_triggers    chan *IncomingTriggers
	incoming_breadcrumbs chan *IncomingBreadcrumbs

	logger *CsvLogger
}

type Agent struct {
	addr              string
	id_to_addr        map[int32]string
	outgoing_triggers chan []Trigger
}

func (s *CoordinatorServer) Init(port string, logfile string) (err error) {
	s.c.Init()
	s.timeout = -60 * time.Second
	s.agents = make(map[string]*Agent)
	s.listen_port = port
	s.incoming_triggers = make(chan *IncomingTriggers, 1000)
	s.incoming_breadcrumbs = make(chan *IncomingBreadcrumbs, 1000)
	if logfile != "" {
		s.logger, err = NewCsvLogger(logfile)
	} else {
		s.logger = nil
	}
	return
}

func (a *Agent) Init(addr string) {
	a.addr = addr
	a.id_to_addr = make(map[int32]string)
	a.outgoing_triggers = make(chan []Trigger, 100)
}

func (s *CoordinatorServer) Run(ctx context.Context) {
	s.ctx = ctx
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		s.runServer(ctx)
		wg.Done()
	}()
	go func() {
		s.runCoordinator(ctx)
		wg.Done()
	}()
	if s.logger != nil {
		cancel := s.logger.Run()
		wg.Wait()
		cancel() // Done like this to ensure everything gets drained properly
		s.logger.AwaitCompletion()
	} else {
		wg.Wait()
	}
}

/* Run the RPC server that receives triggers and breadcrumbs */
func (cs *CoordinatorServer) runServer(ctx context.Context) {
	lis, err := net.Listen("tcp", ":"+cs.listen_port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Println("Listening for agent connections on port", cs.listen_port)
	grpcserver := grpc.NewServer()
	datapb.RegisterCoordinatorServer(grpcserver, cs)

	go func() {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gRPC server")
			grpcserver.GracefulStop()
		}
	}()

	if err := grpcserver.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	} else {
		log.Println("gRPC server finished serving.")
	}
}

func (cs *CoordinatorServer) GetAgent(addr string) *Agent {
	if agent, ok := cs.agents[addr]; ok {
		return agent
	} else {
		var agent Agent
		agent.Init(addr)
		cs.agents[addr] = &agent
		agent.Run(cs.ctx)
		return &agent
	}
}

func (cs *CoordinatorServer) checkExpirations() {
	cs.c.now = time.Now()
	cs.c.checkTraceExpiration(cs.c.now.Add(cs.timeout))
	finished := cs.c.checkTriggerExpiration(cs.c.now.Add(cs.timeout))
	if cs.logger != nil {
		for _, f := range finished {
			cs.logger.Finished <- f
		}
	}
}

func (cs *CoordinatorServer) processTriggersRequest(incoming *IncomingTriggers) {
	cs.c.now = time.Now()
	req := incoming.req

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

	cs.checkExpirations()
	incoming.ret <- nil
}

func (cs *CoordinatorServer) processBreadcrumbRequest(incoming *IncomingBreadcrumbs) {
	cs.c.now = time.Now()
	req := incoming.req

	origin := cs.GetAgent(req.Src)

	// Breadcrumbs are received as IDs; unravel into addr strings
	for _, a := range req.Addresses {
		origin.id_to_addr[a.Id] = a.Addr
	}

	breadcrumbs := make(map[uint64][]string)
	for _, b := range req.Breadcrumbs {
		for _, addr_id := range b.Addrs {
			if addr, ok := origin.id_to_addr[addr_id]; ok {
				breadcrumbs[b.TraceId] = append(breadcrumbs[b.TraceId], addr)
			} else {
				incoming.ret <- fmt.Errorf("Received addr_id %d from %s that hasn't been mapped to an address", addr_id, req.Src)
				return
			}
		}
	}

	// Now process them
	triggers_to_forward := make(map[string][]Trigger)
	for trace_id, addrs := range breadcrumbs {
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

	cs.checkExpirations()
	incoming.ret <- nil
}

/* The "main" thread that receives incoming stuff and sends outgoing stuff */
func (cs *CoordinatorServer) runCoordinator(ctx context.Context) {
	log.Println("CoordinatorServer main goroutine running")
	for {
		select {
		case <-ctx.Done():
			log.Println("CoordinatorServer main goroutine exiting")
			if cs.logger != nil {
				/* Expire everything, so that it flushes to log */
				finished := cs.c.checkTriggerExpiration(time.Now().Add(1 * time.Second))
				for _, f := range finished {
					cs.logger.Finished <- f
				}
			}
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
func (s *CoordinatorServer) LocalTrigger(ctx context.Context, req *datapb.TriggerRequest) (rsp *datapb.TriggerReply, err error) {
	// fmt.Println("Received a local trigger!", in.Src, in.Triggers)
	var incoming IncomingTriggers
	incoming.req = req
	incoming.ret = make(chan error)

	select {
	case s.incoming_triggers <- &incoming:
		err = <-incoming.ret
		if err != nil {
			fmt.Println("Breadcrumbs error:", err.Error())
		}
	default:
		// TODO: counters here
		fmt.Println("LocalTrigger incoming_triggers bottlenecked!")
	}
	rsp = &datapb.TriggerReply{}
	return
}

/* An agent has sent us breadcrumbs */
func (s *CoordinatorServer) Breadcrumbs(ctx context.Context, req *datapb.BreadcrumbsRequest) (rsp *datapb.BreadcrumbsReply, err error) {
	// fmt.Println("Received breadcrumbs!", in.Src, in.Breadcrumbs)
	var incoming IncomingBreadcrumbs
	incoming.req = req
	incoming.ret = make(chan error)

	select {
	case s.incoming_breadcrumbs <- &incoming:
		err = <-incoming.ret
		if err != nil {
			fmt.Println("Breadcrumbs error:", err)
		}
	default:
		// TODO: counters here
		fmt.Println("LocalTrigger incoming_breadcrumbs bottlenecked!")
	}
	rsp = &datapb.BreadcrumbsReply{}
	return
}

func (a *Agent) Run(ctx context.Context) {
	go func() {
		a.AgentLoop(ctx)
	}()
}

/* Connects to an agent in a loop, then sends triggers once connected */
func (a *Agent) AgentLoop(ctx context.Context) {
	fmt.Println("Connecting to agent", a.addr)
	firsttime := true
	for {
		select {
		case <-ctx.Done():
			return
		default:
			break
		}

		conn, err := grpc.Dial(a.addr, grpc.WithInsecure(), grpc.WithTimeout(100*time.Millisecond))
		if err != nil {
			if firsttime {
				fmt.Println("Unable to connect to coordinator, retrying every 2 seconds", a.addr, err)
				firsttime = false
			}
			time.Sleep(time.Duration(2) * time.Second)
			continue
		}
		defer conn.Close()

		rpcclient := datapb.NewAgentClient(conn)

		err = a.ReportTriggers(ctx, rpcclient)
		if err != nil {
			if firsttime {
				fmt.Println("Error with agent", a.addr, err, " -- will retry every 2 seconds")
				firsttime = false
			}
			time.Sleep(time.Duration(2) * time.Second)
			continue
		}

		firsttime = true
	}
}

func (a *Agent) ReportTriggers(ctx context.Context, rpcclient datapb.AgentClient) error {
	for {
		// Accumulate a batch of up to 100 triggers
		var accumulated []Trigger

		// Block waiting for some triggers
		for len(accumulated) == 0 {
			select {
			case triggers := <-a.outgoing_triggers:
				if len(triggers) > 0 {
					accumulated = append(accumulated, triggers...)
				}
			}
		}

		// Now try to batch as many additional triggers as possible (without blocking)
	Accumulation:
		for len(accumulated) < 100 {
			select {
			case <-ctx.Done():
				return nil
			case triggers := <-a.outgoing_triggers:
				if len(triggers) > 0 {
					accumulated = append(accumulated, triggers...)
				}
			default:
				break Accumulation
			}
		}

		// Send them
		err := a.doSend(rpcclient, accumulated)

		if err != nil {
			return err
		}
	}
}

/* Send a batch of local triggers to the coordinator */
func (a *Agent) doSend(rpcclient datapb.AgentClient, triggers []Trigger) error {
	var request datapb.TriggerRequest
	for _, trigger := range triggers {
		var t datapb.Trigger
		t.QueueId = int32(trigger.id.queue_id)
		t.BaseTraceId = trigger.id.base_trace_id
		t.TraceIds = trigger.trace_ids
		request.Triggers = append(request.Triggers, &t)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := rpcclient.RemoteTrigger(ctx, &request)

	return err
}

func (a *Agent) SendTriggers(triggers []Trigger) {
	// fmt.Println("Forwarding triggers!", a.addr, triggers)
	select {
	case a.outgoing_triggers <- triggers:
		break
	default:
		// TODO: counters here
		fmt.Println("Agent SendTriggers bottlenecked!", a.addr)
	}
}
