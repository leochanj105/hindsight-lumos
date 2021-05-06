#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <argp.h>
#include <pthread.h>

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
  {"duration", 'd', "NUM", 0, "Duration in seconds before exiting. 0 to run forever"},
  {"output",   'o', "FILE", 0, "Output stats to FILE" },
  { 0 }
};

static inline unsigned long long getticks(void)
{
    unsigned int lo, hi;

    // RDTSC copies contents of 64-bit TSC into EDX:EAX
    asm volatile("rdtsc" : "=a" (lo), "=d" (hi));
    return (unsigned long long)hi << 32 | lo;
}

struct arguments {
  int num_threads;
  size_t buffer_size;
  size_t buffer_count;
  size_t payload_size;
  int tracepoints_per_request;
  float trigger_probability;
  uint64_t duration;
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
    case 'd':
      arguments->duration = atoll(arg);
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

void init_hindsight_client(struct arguments *arguments) {
    HindsightConfig conf;
    conf.pool_capacity = arguments->buffer_count;
    conf.buffer_size = arguments->buffer_size;
    conf.breadcrumbs_capacity = conf.pool_capacity;
    conf.triggers_capacity = conf.pool_capacity;
    conf.address = malloc(32 * sizeof(char));

    hindsight_init_with_config(arguments->process_name, conf);
}

typedef struct client_args {
    volatile int *alive;
    int client_id;
    struct arguments* arguments;
} client_args;

void client_thread_main(volatile int *alive, 
        int client_id, struct arguments *arguments) {
    printf("Client %d started\n", client_id);

    size_t payload_src_size = arguments->payload_size;
    char payload[payload_src_size];

    uint64_t trace_id = 1000000LL * client_id;
    int tracepoints_per_request = arguments->tracepoints_per_request;
    while (*alive) {
        hindsight_begin(++trace_id);
        for (int i = 0; i < tracepoints_per_request; i++) {
            hindsight_tracepoint(payload, payload_src_size);
        }
        hindsight_end();
    }
    printf("Client ended\n");
}

void* run_client_thread(void *vargp) {
    client_args* args = (client_args*) vargp;
    client_thread_main(args->alive, args->client_id, args->arguments);
    return 0;
}

void run_clients(struct arguments *arguments) {
    volatile int alive = 1;
    printf("Running clients\n");
    pthread_t threads[arguments->num_threads];
    client_args args[arguments->num_threads];
    for (int i = 0; i < arguments->num_threads; i++) {
        args[i].alive = &alive;
        args[i].client_id = i;
        args[i].arguments = arguments;
        pthread_create(&threads[i], NULL, run_client_thread, (void*) &args[i]);
    }


    uint64_t end = nanos() + arguments->duration * 1000000000LL;
    if (arguments->duration == 0) {
        end = -1;
    }

    while (true) {
        uint64_t now = nanos();
        if (now > end) {
            break;
        }
        usleep(1000000);
    }
    alive = 0;
    for (int i = 0; i < arguments->num_threads; i++) {
        pthread_join(threads[i], NULL);
    }
    printf("Clients complete.\n");
}


int main (int argc, char **argv) {
  struct arguments arguments;

  /* Default values. */
  arguments.num_threads = 1;
  arguments.buffer_size = 4096;
  arguments.buffer_count = 25000; // Default 100MB pool
  arguments.payload_size = 400;
  arguments.tracepoints_per_request = 100;
  arguments.trigger_probability = 0;
  arguments.duration = 0;


  arguments.process_name = 0;
  arguments.output_file = "-";

  /* Parse our arguments; every option seen by parse_opt will
     be reflected in arguments. */
  argp_parse (&argp, argc, argv, 0, 0, &arguments);

  printf("name=%s\nbuffer_size=%ld\nbuffer_count=%ld\nnum_threads=%d\npayload_size=%ld\ntp_per=%d\ntrigger=%.3f\nduration=%lu\n-------\n",
            arguments.process_name,
            arguments.buffer_size, arguments.buffer_count,
            arguments.num_threads, arguments.payload_size,
            arguments.tracepoints_per_request,
            arguments.trigger_probability,
            arguments.duration);

  init_hindsight_client(&arguments);
  printf("------\n");

  run_clients(&arguments);

  exit (0);
}