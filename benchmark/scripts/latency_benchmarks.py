import json
import subprocess
import signal
import argparse
import time
import numpy as np
import math
import os
import pandas as pd

parser = argparse.ArgumentParser(description='Run multiple benchmarks')
parser.add_argument("out", metavar="OUT", type=str, help="Output directory")
parser.add_argument("-s", "--from_scratch", action='store_true', help='If true, overwrite existing experiment results')
parser.add_argument("-r", "--results", action='store_true', help='Process results only')

shmname = "multi"
duration = 60 # Duration per run
# threads = list(range(1,33))
threads = [1,2,4,6,8,12,16,20,24,28,32]
buffer_size = 32768
trace_sizes = [1024, 4096, 16384, 65536]
payload_sizes = [4,8,16,32,64,128,256,512,1024,2048,4096]

def make_cmd(args, threads, outdir, trace_size, payload_size):
    tracepoints = int(trace_size / payload_size)

    return [str(v) for v in [
        "python3", "run_benchmark2.py",
        "--threads", threads,
        "--duration", duration,
        "--buffer_size", buffer_size,
        "--payload_size", payload_size,
        "--tracepoints", tracepoints,
        "--silent",
        "--header",
        shmname,
        outdir
    ]]

def make_experiments(args):
    exps = []
    for t in threads:
        for trace_size in trace_sizes:
            for payload_size in payload_sizes:
                outdir = "%s/%d_%d_%dthreads" % (args.out, trace_size, payload_size, t)
                exps.append({
                    "name": "%dthreads" % t,
                    "cmd": make_cmd(args, t, outdir, trace_size, payload_size),
                    "threads": t,
                    "trace_size": trace_size,
                    "payload_size": payload_size,
                    "outdir": outdir,
                    "exists": os.path.exists(outdir)
                })
    return exps


def run_experiments(args):
    exps = make_experiments(args)

    to_run = []
    to_skip = []
    if args.results:
        to_skip = exps
    elif args.from_scratch:
        to_run = exps
    else:
        to_run = [exp for exp in exps if not exp["exists"]]
        to_skip = [exp for exp in exps if exp["exists"]]

    if len(to_skip) > 0:
        print("Skipping %d experiments" % len(to_skip))
    if len(to_run) > 0:
        print("Running %d experiments (%.2f hours):" % (len(to_run), (len(to_run) * duration)/3600))
        print("  threads %s" % threads)
        print("  trace_sizes %s" % trace_sizes)
        print("  payload_sizes %s" % payload_sizes)
        print("Press <return> to continue or CTRL-C to abort")
        input()

    for i, exp in enumerate(to_run):
        print("Running %d/%d" % (i+1, len(to_run)))
        print(exp["cmd"])
        runner = subprocess.Popen(exp["cmd"])
        runner.wait()
    
    process_results(exps)
    
def process_results(exps):
    df = None
    for exp in exps:
        try:
            filename = "%s/benchmark.out" % exp["outdir"]
            with open(filename, "r") as f:
                lines = f.readlines()
            headerline = [l.strip() for l in lines if l.startswith("headers:")][0]
            datalines = [l.strip() for l in lines if l.startswith("data:")]
            datalines = datalines[int(len(datalines)/2):len(datalines)-1]
            headers = headerline.split("\t")[1:]
            data = [l.split("\t")[1:] for l in datalines]
            rows = [dict(zip(headers, d)) for d in data]
            for row in rows:
                row["threads"] = exp["threads"]
                row["trace_size"] = exp["trace_size"]
                row["payload_size"] = exp["payload_size"]
            if df is None:
                df = pd.DataFrame(rows)
            else:
                df = df.append(rows)
        except Exception as e:
            print("Skipping results for %s: %s" % (exp["outdir"], e))
    
    if df is None:
        print("No results to process")
        return

    df = df.apply(pd.to_numeric)

    columns = [
        "begin", "tracepoint",
        "filtertriggers", "categorytriggers", "p99triggers", "p999triggers",
        "p9999triggers", "percentiletriggersets", "end"
    ]

    means = df.groupby(["threads", "trace_size", "payload_size"])[columns].mean()
    means.to_csv("%s/latencies.out" % args.out)

if __name__ == '__main__':
    args = parser.parse_args()
    run_experiments(args)
