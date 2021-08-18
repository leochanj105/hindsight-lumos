#include <stdio.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <assert.h>
#include <string.h>

#include "hindsight.h"

Hindsight hindsight;
BufManager* mgr;
__thread TraceState hindsight_tls = {false};

void hindsight_print_config(HindsightConfig* conf) {
    printf("Hindsight Config:\n");
    printf("  Buffer pool cap=%ld buf_length=%ld\n", conf->pool_capacity, conf->buffer_size);
    printf("  Service addr=%s\n", conf->address);
    printf("  Queue sizes breadcrumbs_cap=%ld triggers_cap=%ld\n", conf->breadcrumbs_capacity, conf->triggers_capacity);
}

HindsightConfig hindsight_load_config(const char* fname) {
    // Initialize config with defaults
    HindsightConfig conf;
    conf.pool_capacity = -1; // size_t doesn't have negatives but we won't use comparisons
    conf.buffer_size = -1;
    conf.breadcrumbs_capacity = -1;
    conf.triggers_capacity = -1;
    conf.payload = 1;
    conf.sample_rate = 1;  

    // Addr in the conf file is specified as separate address and port strings
    char* conf_addr = (char*) malloc(32 * sizeof(char));
    char* conf_port = (char*) malloc(32 * sizeof(char));
    memset(conf_addr, 0, 32*sizeof(char));
    memset(conf_port, 0, 32*sizeof(char));
    
    // Open the specified file, with defaults as backup
    FILE* config_file;
    config_file = fopen(fname,"r");
    if (config_file == NULL) {
        config_file = fopen(HINDSIGHT_DEFAULT_CONFIG,"r");
    }

    // Read the config
    char* line = NULL;
    ssize_t read;
    size_t len = 0;
    while((read = getline(&line, &len, config_file)) != -1) {
        char* temp = strchr(line, '\n');
        int index = (int)(temp - line);

        char* new_line = malloc(sizeof(char)*32);
        if (index == strlen(line)-1) {
            strncpy(new_line, line, index);
        } else {
            strncpy(new_line, line, strlen(line));
        }

        char* var = malloc(sizeof(char)*32);
        memset(var, 0, 32*sizeof(char));
        char* value = malloc(sizeof(char)*32);
        memset(value, 0, 32*sizeof(char));
        sscanf(new_line, "%s %s", var, value);

        if (!strcmp(var, "cap")) {
            conf.pool_capacity = atoi(value);
        }

        if (!strcmp(var, "buf_length")) {
            conf.buffer_size = atoi(value);
        }

        if (!strcmp(var, "addr")) {
            conf_addr = value;
        }

        if (!strcmp(var, "port")) {
            conf_port = value;
        }       

        if (!strcmp(var, "breadcrumbs_cap")) {
            conf.breadcrumbs_capacity = atoi(value);
        }       

        if (!strcmp(var, "triggers_cap")) {
            conf.triggers_capacity = atoi(value);
        }       

        if (!strcmp(var, "payload")) {
            conf.payload = atoi(value);
        }       

        if (!strcmp(var, "sample_rate")) {
            conf.sample_rate = atoi(value);
        }
    }
    fclose(config_file);

    if (line) free(line);

    
    // Addr in the conf struct is a single string of address:port
    conf.address = (char*) malloc(32 * sizeof(char));
    memset(conf.address, 0, 32*sizeof(char));

    strcpy(conf.address, conf_addr);
    strcat(conf.address, ":");
    strncat(conf.address, conf_port, 4);

    if (conf.pool_capacity == -1) conf.pool_capacity = 1;
    if (conf.buffer_size == -1) conf.buffer_size = 1;
    if (conf.breadcrumbs_capacity == -1) conf.breadcrumbs_capacity = conf.pool_capacity;
    if (conf.triggers_capacity == -1) conf.triggers_capacity = conf.pool_capacity;

    return conf;
}

void hindsight_init(const char* service_name) {
    // Load Hindsight conf for this service
    char config_fname[64];
    strcpy(config_fname, "/etc/hindsight_conf/");
    strcat(config_fname, service_name);
    strcat(config_fname, ".conf");
    hindsight_init_with_config(service_name, hindsight_load_config(config_fname));
}

void hindsight_init_with_config(const char* service_name, HindsightConfig config) {
    hindsight.config = config;
    hindsight_print_config(&hindsight.config);

    // Create pools and queues
    hindsight.mgr = bufmanager_init(
        service_name,
        hindsight.config.pool_capacity,
        hindsight.config.buffer_size);

    hindsight.breadcrumbs = breadcrumbs_init(
        service_name, 
        hindsight.config.breadcrumbs_capacity);

    hindsight.triggers = triggers_init(
        service_name,
        hindsight.config.triggers_capacity);

    mgr = &hindsight.mgr;

    tail_init();
}

void hindsight_begin(uint64_t trace_id) {
    tracestate_begin(&hindsight_tls, mgr, trace_id);
}

void hindsight_begin_sampling(uint64_t trace_id) {
    tracestate_begin_sampling(&hindsight_tls, mgr, trace_id, hindsight.config.sample_rate);
}

void hindsight_end() {
    tracestate_end(&hindsight_tls, mgr);
}

void hindsight_tracepoint(char* buf, size_t buf_size) {
    if (tracestate_try_write(&hindsight_tls, buf, buf_size)) return;
    tracestate_write(&hindsight_tls, mgr, buf, buf_size);
}

void hindsight_tracepoint_write(size_t write_size, char** dst, size_t* dst_size) {
    tracestate_write_data(&hindsight_tls, mgr, 
        write_size, dst, dst_size);
}

void hindsight_breadcrumb(const char* addr) {
    breadcrumbs_add(&hindsight.breadcrumbs, hindsight_tls.header.trace_id, addr);
}

void hindsight_forward_breadcrumb(const char* addr) {
    breadcrumbs_add_forward(&hindsight.breadcrumbs, hindsight_tls.header.trace_id, addr);
}

void hindsight_trigger(int trigger_id) {
    triggers_fire(&hindsight.triggers, trigger_id, hindsight_tls.header.trace_id);
}

void hindsight_trigger_manual(uint64_t trace_id, int trigger_id) {
    triggers_fire(&hindsight.triggers, trigger_id, trace_id);   
}

uint64_t hindsight_get_traceid() {
    return hindsight_tls.header.trace_id;
}

char* hindsight_get_local_address() {
    return hindsight.config.address;
}

char* hindsight_serialize() {
    return hindsight_get_local_address();
}

void hindsight_deserialize(char* baggage) {
    hindsight_breadcrumb(baggage);
}

int hindsight_payload() {
    return hindsight.config.payload;
}

int hindsight_sample_rate() {
    return hindsight.config.sample_rate;
}

int hindsight_null_buffer_count() {
    return hindsight_tls.header.null_buffer_count;
}