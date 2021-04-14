# Hindsight Tracing System

## Run Hindsight on local

### *Environmental Setup*
* gcc
* golang, with several packages
```
go get golang.org/x/net/context golang.org/x/exp/mmap google.golang.org/grpc
```
* you may need to add agent dir to $GOPATH

### *Testing Hindsight*
1. Make hindsight client
```
cd $hindsight_dir/client && make
```
2. Run client *first*
```
bin/client_test
```
3. Run agent (on local mode)
```
cd $hindsight_dir/agent && go run main.go -serv=client -report=false
```
4. Replace *client* with *multi_client* to test multi thread environments

## Hindsight Instrumentation APIs
```
#define TRACEINIT(x) 		trace_init(x) 				// init with service name (e.g. "serv")

#define TRACEBEGIN(x) 		trace_begin(x,y,z) 			// start trace with request_id 
#define TRACEEND()  		trace_end() 				// end trace
#define TRACEPOINT(x,y) 	tracepoint(x,y) 			// add tracepoint, with tracepoint_id (currently not used) and payload
#define TRIGGER(x) 			trigger(x) 					// trigger with request_id or tracepoint_id
#define TRACEBREADCRUMB(x) 	trace_add_breadcrumb(x) 	// add breadcrumb (e.g. "127.0.0.1:5050")

#define SERIALIZE() 		serialize() 				// get local *AGENT* address
#define DESERIALIZE() 		deserialize() 				// get parent *AGENT* address
```