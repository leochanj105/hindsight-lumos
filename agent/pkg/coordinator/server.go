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
}

type Agent struct {
	addr string
}

func (s *CoordinatorServer) Init(port string) {
	s.c.Init()
	s.agents = make(map[string]*Agent)
	s.listen_port = port
}

func (s *CoordinatorServer) Run(ctx context.Context) {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		s.runServer()
		wg.Done()
	}()
	wg.Wait()
}

/* Run the server that receives triggers and breadcrumbs */
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

/* An agent has sent us a trigger */
func (s *CoordinatorServer) LocalTrigger(ctx context.Context, in *datapb.TriggerRequest) (*datapb.TriggerReply, error) {
	fmt.Println("Received a local trigger!", in.Triggers)
	return &datapb.TriggerReply{}, nil
}

/* An agent has sent us breadcrumbs */
func (s *CoordinatorServer) Breadcrumbs(ctx context.Context, in *datapb.BreadcrumbsRequest) (*datapb.BreadcrumbsReply, error) {
	fmt.Println("Received breadcrumbs!", in.Breadcrumbs)
	return &datapb.BreadcrumbsReply{}, nil
}

// func initAgent(addr string) *Agent {

// }

// func (a *Agent) Run() {

// }
