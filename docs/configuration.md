# Configuration

For environment setup, see [environment](environment.md)

The default Hindsight configuration can be found in `conf/default.conf`:

```
cap 10000
buf_length 32768
addr 127.0.0.1
port 5050
lc_addr 127.0.0.1
lc_port 5252
r_addr 127.0.0.1
r_port 5253
retroactive_sampling_percentage 1.0
head_sampling_probability 0.0
```

There are some other configuration values that are used for experiments and are here for convenience until they can be refactored:

```
payload 1000
```

**Reminder:** there are three categories of process that use Hindsight: clients, which are spread across many machines; agents, also spread across machines -- agents are separate go processes that run side-by-side with clients; and a single centralized log collector.

* `cap`: The default number of buffers in Hindsight's buffer pool
* `buf_length`: The default buffer size, in bytes, of buffers in Hindsight's buffer pools
  * *The size, in bytes, of Hindsight's buffer pool is `cap * buf_length`*
* `addr`: The hostname or IP address of the host loading this config file
* `port`: The port to be used by the agent
* `lc_addr`: The hostname or IP address of the log collector
* `lc_port`: The port of the log collector

## How is config used?

* Each agent will read the config and create a gRPC server listening on `addr:port`; the log collector can contact the agent using this gRPC server
* Each client will set up its own shared memory using `cap` and `buf_length`
* Each client will pass `addr:port` as a breadcrumb inside any propagated contexts

## Process names

Hindsight clients and agents identify each other with process names:
* Within a client, when Hindsight is initialized, `hindsight_init(char* process_name)` receives the process name; this must be provided by the caller
* With the co-located agent, the process name is passed as a command line argument: `go run cmd/agent2/main.go --serv process_name`
* Both the client and the agent will load the config file from `/etc/hindsight_conf/{process_name}.conf`, e.g. `/etc/hindsight_conf/my_process.conf`
  * If no config file exists, it will load `default.conf`; this is fine for a single-node setup
* The shared-memory files will be prefixed with the `process_name`; if the agent is given a different process_name than the client, then they won't find each other.

## Writing a configuration

For a node named `datanode`, copy default.conf into a new file `datanode.conf`.  Update the `addr` and `lc_addr` with appropriate values, e.g.
```
cd client
cp conf/default.conf conf/datanode.conf
```
Then edit `conf/datanode.conf`:
```
cap 1000000
buf_length 50
addr 196.168.0.101
port 5050
lc_addr 192.168.0.99
lc_port 5252
```
Then
```
sudo make install
```
This will copy all config files to `/etc/hindsight_conf`.  e.g.
```
less /etc/hindsight_conf/datanode.conf
```
