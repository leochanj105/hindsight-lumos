# Running Hindsight's Collector

Hindsight's collector is a basic "trace collection backend" service.  It has minimal features -- it simply receives data sent by different Hindsight agents.

Running the agent:

```
go run cmd/collector/main.go
```

Expected output:

```
Running coordinator
  port=5253 (default)
Collector listening on TCP port 5253
1.38 MB/s
0.23 MB/s
0.53 MB/s
```

If there are agents generating data, the collector will periodically print the throughput of received data as shown above. 

# Configuring the Collector

By default Hindsight's collector will listen on port `5253`.  You can change the port of the collector with the `-port` flag.  

See the full collector options with the `--help` flag:

```
Usage of /tmp/go-build229934906/b001/exe/main:
  -port r_port
        Collector port.  If not specified, uses r_port from the legacy config lc.conf file, or 5253 as a backup
```

# Configuring Agents to Point to the Collector

Hindsight agents report buffers to the collector, and thus they need the address of the collector.  If the collector isn't running or if it is misconfigured, then the agent will periodically retry connecting in the background and data will not be reported.  For example you will see the following output when running an agent:

```
Unable to connect to reporting backend, retrying every 2 seconds 127.0.0.1:5253 dial tcp 127.0.0.1:5253: connect: connection refused
```

### Configuring Agents via the Command Line

The collector address can be given to agents with the `-r` flag, e.g.

```
go run cmd/agent2/main.go -r 127.0.0.1:5253
```

Alternatively, it can be configured in the conf file with the `r_addr` and `r_port` keys e.g.

`my_agent.conf`:
```
r_addr 127.0.0.1
r_port 5253
```

```
go run cmd/agent2/main.go --serv my_agent
```