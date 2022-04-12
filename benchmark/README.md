# Building

Make sure you have built and installed Hindsight `client`

```
mkdir build
cd build
cmake ..
make -j
```

# Running

```
cd build
./benchmark
```

# Options

```
Usage: benchmark [OPTION...] PROCNAME
Simple hindsight benchmarking program.  PROCNAME is required for
hindsight_init

  -c, --buffer_count=NUM     Number of buffers in pool
  -d, --duration=NUM         Duration in seconds before exiting. 0 to run
                             forever
  -H, --headsampling=NUM     Head-based sampling probability between 0 and 1,
                             default 0, float
  -n, --tracepoints=NUM      Number of tracepoints per trace
  -o, --output=FILE          Output stats to FILE
  -p, --trigger=NUM          Trigger probability, default 0, float
  -R, --retroactive=NUM      Retroactive sampling percentage between 0 and 1,
                             default 1, float
  -s, --buffer_size=NUM      Buffer size in bytes
  -t, --threads=NUM          Number of benchmark threads to run
  -w, --payload_size=NUM     Payload size written by each tracepoint
  -?, --help                 Give this help list
      --usage                Give a short usage message
  -V, --version              Print program version

Mandatory or optional arguments to long options are also mandatory or optional
for any corresponding short options.

Report bugs to <bug-gnu-utils@gnu.org>.
```