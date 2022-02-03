# Running Hindsight's Coordinator

Hindsight's coordinator is responsible for disseminating breadcrumbs and triggers between agents.

Running the coordinator:

```
go run cmd/coordinator/main.go
```

Expected output:

```
Running coordinator
  port=5252 (command line)
CoordinatorServer main goroutine running
Listening for agent connections on port 5252
Forwarding triggers! 127.0.0.1:5053 [{{10 2531011} [2531011]} {{10 541671788154} [541671788154]} {{10 115924804400733013} [115924804400733013]}]
Connecting to agent 127.0.0.1:5053
Forwarding triggers! 127.0.0.1:5053 [{{10 16991129148439470276} [16991129148439470276]}]
Forwarding triggers! 127.0.0.1:5053 [{{10 8096914980992404599} [8096914980992404599]}]
Forwarding triggers! 127.0.0.1:5053 [{{10 13267775073337824606} [13267775073337824606]}]
```

**TODO: suppress verbose trigger printing, print stats instead **

# Configuring the Coordinator

By default Hindsight's coordinator will listen on port `5252`.  You can change the port of the coordinator with the `-port` flag.  

See the full coordinator options with the `--help` flag:

```Usage of /tmp/go-build1913113332/b001/exe/main:
  -port lc_port
        Coordinator port.  If not specified, uses lc_port from the legacy config lc.conf file. (default "5252")
```

# Configuring Agents to Point to the Coordinator

Hindsight agents report breadcrumbs and triggers to the coordinator, and thus they need the address of the coordinator.  If the coordinator isn't running or if it is misconfigured, then the agent will periodically retry connecting in the background and data will not be reported.  For example you will see the following output when running an agent:

```
Error in TriggersLoop: rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp 127.0.0.1:5252: connect: connection refused"  -- will retry every 2 seconds
Error in BreadcrumbsLoop: rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp 127.0.0.1:5252: connect: connection refused"  -- will retry every 2 seconds
```

This error will only show up lazily when we attempt to report the first triggers and breadcrumbs to the coordinator.

### Configuring Agents via the Command Line

The coordinator address can be given to agents with the `-lc` flag, e.g.

```
go run cmd/agent2/main.go -lc 127.0.0.1:5252
```

Alternatively, it can be configured in the conf file with the `lc_addr` and `lc_port` keys e.g.

`my_agent.conf`:
```
lc_addr 127.0.0.1
lc_port 5252
```

```
go run cmd/agent2/main.go --serv my_agent
```