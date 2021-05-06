#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <argp.h>

#include "buffer.h"
#include "tracer.h"
#include "hindsight.h"
#include "agentapi.h"
#include "common.h"
#include <time.h>

const char *argp_program_version = "argp-ex3 1.0";
const char *argp_program_bug_address = "<bug-gnu-utils@gnu.org>";
static char doc[] = "Simple hindsight benchmarking program.  PROCNAME is required for hindsight_init";
static char args_doc[] = "PROCNAME";

static struct argp_option options[] = {
  {"threads",  't', "NUM",  0,  "Number of benchmark threads to run" },
  {"buffer_size",  's', "NUM",  0,  "Buffer size in bytes" },
  {"buffer_count",  'c', "NUM",  0,  "Number of buffers in pool" },
  {"tracepoints", 'n', "NUM", 0, "Number of tracepoints per trace"},
  {"trigger", 'p', "NUM", 0, "Trigger probability, default 0, float"},
  {"output",   'o', "FILE", 0, "Output stats to FILE" },
  { 0 }
};

struct arguments {
  int num_threads;
  size_t buffer_size;
  size_t buffer_count;
  size_t payload_size;
  int tracepoints_per_request;
  float trigger_probability;
  char* output_file;
  char* process_name;
};

static error_t parse_opt (int key, char *arg, struct argp_state *state) {
  struct arguments *arguments = state->input;

  switch (key)
    {
    case 't':
      arguments->num_threads = atoi(arg);
      break;
    case 's':
      arguments->buffer_size = atoll(arg);
      break;
    case 'c':
      arguments->buffer_count = atoll(arg);
      break;
    case 'n':
      arguments->tracepoints_per_request = atoi(arg);
      break;
    case 'p':
      arguments->trigger_probability = atof(arg);
      break;
    case 'o':
      arguments->output_file = arg;
      break;

    case ARGP_KEY_ARG:
      if (state->arg_num >= 1)
        /* Too many arguments. */
        argp_usage (state);

      arguments->process_name = arg;

      break;

    case ARGP_KEY_END:
      if (state->arg_num < 1)
        /* Not enough arguments. */
        argp_usage (state);
      break;

    default:
      return ARGP_ERR_UNKNOWN;
    }
  return 0;
}


static struct argp argp = { options, parse_opt, args_doc, doc };

int main (int argc, char **argv) {
  struct arguments arguments;

  /* Default values. */
  arguments.num_threads = 1;
  arguments.buffer_size = 4096;
  arguments.buffer_count = 25000; // Default 100MB pool
  arguments.payload_size = 400;
  arguments.tracepoints_per_request = 100;
  arguments.trigger_probability = 0;


  arguments.process_name = 0;
  arguments.output_file = "-";

  /* Parse our arguments; every option seen by parse_opt will
     be reflected in arguments. */
  argp_parse (&argp, argc, argv, 0, 0, &arguments);

  printf("name=%s\nbuffer_size=%ld\nbuffer_count=%ld\nnum_threads=%d\npayload_size=%ld\ntp_per=%d\ntrigger=%.3f\n",
  			arguments.process_name,
  			arguments.buffer_size, arguments.buffer_count,
  			arguments.num_threads, arguments.payload_size,
  			arguments.tracepoints_per_request,
  			arguments.trigger_probability);

  exit (0);
}