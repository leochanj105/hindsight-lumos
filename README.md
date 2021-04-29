# Hindsight Tracing System

## Run Hindsight on local

### *Environmental Setup*
* gcc
* golang 1.11 or higher (to support go modules)
* you may need to add agent dir to `$GOPATH`
* Hindsight's agent uses cgo::
```
export CGO_LDFLAGS_ALLOW=".*"
export GOMAXPROCS=10
```

### First build
0. Build the client.
```
cd client
make
```
1. Run single-process tests
```
bin/queue_test
bin/buffer_test
```
2. Run multi-process tests
Open two terminals.  In the first, run:
```
bin/hindsight_test x
```
Then, in the second, run:
```
bin/hindsight_test
```
You will see some outputs, such as pool sizes and shm files used.  Then you will see output such as:

Terminal 1:
```
Throughput: 2435647
Throughput: 2436865
```

Terminal 2:
```
Tracepoints 22486887 - Pool: 0 0 - NULL 5846590 5846589
Tracepoints 13126025 - Pool: 1725364 1725363 - NULL 1687401 1687402
Tracepoints 9351492 - Pool: 2431388 2431388 - NULL 0 0
```

The first terminal acts as the "client" -- writing to Hindsight's client API

The second terminal acts as a simple "agent" -- receiving written trace data and recycling buffers

Note: 


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
