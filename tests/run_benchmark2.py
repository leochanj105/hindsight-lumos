import json
import subprocess
import signal
import argparse
import time
import numpy as np
import math
import os
import threading

parser = argparse.ArgumentParser(description='Run a hindsight benchmark')
parser.add_argument("name", metavar="SERVNAME", type=str, help="Service name")
parser.add_argument("out", metavar="OUT", type=str, help="Output directory")
parser.add_argument('-t', "--threads", metavar="NUM", type=int, default="1", help='Number of threads')
parser.add_argument('-s', "--buffer_size", metavar="NUM", type=int, default="4096", help='Buffer size')
parser.add_argument('-c', "--buffer_count", metavar="NUM", type=int, default="25000", help='Buffer Count')
parser.add_argument('-w', "--payload_size", metavar="NUM", type=int, default="1000", help='Payload size')
parser.add_argument('-n', "--tracepoints", metavar="NUM", type=int, default="100", help='Number of tracepoints per trace')
parser.add_argument('-p', "--trigger", metavar="NUM", type=float, default="0", help='Trigger probability')
parser.add_argument('-H', "--headsampling", metavar="NUM", type=float, default="0", help='Head-based sampling probability')
parser.add_argument('-R', "--retroactive", metavar="NUM", type=float, default="1", help='Retroactive tracing probability')
parser.add_argument('-d', "--duration", metavar="NUM", type=int, default="60", help='Experiment duration')
parser.add_argument('-silent', "--silent", action='store_true', help='Prompt before proceeding')
parser.add_argument("--header", action='store_true', help='If set, each tracepoint call will write a TraceEvent header from Hindsights OT library')


import pathlib
from pathlib import Path
def find_models(basedir):
    paths = list(Path(basedir).rglob("model.clockwork_params"))
    paths = [str(p)[:-17] for p in paths]
    return paths

def mkdirs(args):
    if not os.path.isdir(args.out):
        print("out dir %s does not exist." % args.out)
        os.makedirs(args.out)


def reset_shm(args):
    files = ["complete_queue", "triggers_queue", "pool", "breadcrumbs_queue", "available_queue"]
    for file in files:
        cmd = ["rm", "-v", "/dev/shm/%s__%s" % (args.name, file)]
        child = subprocess.Popen(cmd)
        child.wait()
    print("Reset shm")

files = []

def run_client(args):
    cmd_args = ["./benchmark"]
    cmd_args += ["--threads", args.threads]
    cmd_args += ["--buffer_size", args.buffer_size,
        "--buffer_count", args.buffer_count,
        "--payload_size", args.payload_size,
        "--tracepoints", args.tracepoints,
        "--trigger", args.trigger,
        "--duration", args.duration,
        "--headsampling", args.headsampling,
        "--retroactive", args.retroactive]
    if args.header:
        cmd_args += ["--header"]
    cmd_args += [args.name]
    cmd = [str(v) for v in cmd_args]

    print(" ".join(cmd))

    output = "%s/benchmark.out" % args.out
    f = open(output, "w")
    files.append(f)

    p = subprocess.Popen(cmd, stdout=f, stderr=f, cwd="../benchmark/build", preexec_fn=os.setsid)
    return p

def run_agent(args):
    cmd = ["go", "run", "cmd/agent2/main.go", "--serv", args.name]
    print(" ".join(cmd))

    output = "%s/agent.out" % args.out
    f = open(output, "w")
    files.append(f)

    p = subprocess.Popen(cmd, stdout=f, stderr=f, cwd="../agent", preexec_fn=os.setsid)
    return p


def run(args):
    if not args.silent:
        print("Delete shm files for %s? Press <return> to continue or CTRL-C to abort" % args.name)
        input()

    mkdirs(args)
    reset_shm(args)

    exit_flag = threading.Event()

    def signal_handler(sig, frame):
        print("Killing experiment processes...")
        exit_flag.set()
    signal.signal(signal.SIGINT, signal_handler)

    client = run_client(args)
    agent = run_agent(args)

    exit_flag.wait(args.duration+1)

    os.killpg(os.getpgid(client.pid), signal.SIGINT)
    os.killpg(os.getpgid(agent.pid), signal.SIGINT)
    client.wait()
    agent.wait()

    for f in files:
        f.close()

if __name__ == '__main__':
    args = parser.parse_args()
    exit(run(args))
