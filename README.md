# Hindsight Tracing System

## Run Hindsight on local

### *Environmental Setup*
* gcc
* golang, with several packages
```
go get golang.org/x/net/context golang.org/x/exp/mmap google.golang.org/grpc
```
* you may need to add agent dir to $GOPATH
* allow cgo:
```
export CGO_LDFLAGS_ALLOW=".*"
```

### *Testing Hindsight*
0. Write configuration files. This is to define memory cap and buffer length, and agent/log collector addresses. Config file should be named by *[serv_name].conf* under *conf/*. Hindsight will use *default.conf* which is fine for single agent/non-report modes.

1. Make hindsight client and install (especially to install .conf files to /etc/hindsight_conf/)
```
cd $hindsight_dir/client && make && (sudo) make install
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