#include <stdio.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <assert.h>
#include <string.h>

#include "hindsight.h"

Hindsight hindsight;
__thread TraceState hindsight_tls = {false};

void hindsight_print_config(HindsightConfig* conf) {
	printf("Hindsight Config:\n");
	printf("  Buffer pool cap=%ld buf_length=%ld\n", conf->pool_capacity, conf->buffer_size);
	printf("  Service addr=%s port=%s\n", conf->address, conf->port);
	printf("  Queue sizes breadcrumbs_cap=%ld triggers_cap=%ld\n", conf->breadcrumbs_capacity, conf->triggers_capacity);
}

HindsightConfig hindsight_load_config(const char* fname) {
	// Initialize config with defaults
	HindsightConfig conf;
	conf.pool_capacity = -1; // size_t doesn't have negatives but we won't use comparisons
	conf.buffer_size = -1;
	conf.breadcrumbs_capacity = -1;
	conf.triggers_capacity = -1;
	conf.address = malloc(32 * sizeof(char));
	conf.port = malloc(32 * sizeof(char));
	memset(conf.address, 0, 32*sizeof(char));
	memset(conf.port, 0, 32*sizeof(char));
	
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

		char* new_line = malloc(sizeof(char)*20);
		if (index == strlen(line)-1) {
			strncpy(new_line, line, index);
		} else {
			strncpy(new_line, line, strlen(line));
		}

		char* var = malloc(sizeof(char)*20);
		char* value = malloc(sizeof(char)*20);
		sscanf(new_line, "%s %s", var, value);

		if (!strcmp(var, "cap")) {
			conf.pool_capacity = atoi(value);
		}

		if (!strcmp(var, "buf_length")) {
			conf.buffer_size = atoi(value);
		}

		if (!strcmp(var, "addr")) {
			conf.address = value;
		}

		if (!strcmp(var, "port")) {
			conf.port = value;
		}		

		if (!strcmp(var, "breadcrumbs_cap")) {
			conf.breadcrumbs_capacity = atoi(value);
		}		

		if (!strcmp(var, "triggers_cap")) {
			conf.triggers_capacity = atoi(value);
		}		
	}
	fclose(config_file);

	if (line) free(line);

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
}

void hindsight_begin(uint64_t trace_id) {
	tracestate_begin(&hindsight_tls, &hindsight.mgr, trace_id);
}

void hindsight_end() {
	tracestate_end(&hindsight_tls, &hindsight.mgr);
}

void hindsight_tracepoint(char* buf, size_t buf_size) {
	tracestate_write(&hindsight_tls, &hindsight.mgr, buf, buf_size);
}

void hindsight_tracepoint_write(size_t write_size, char** dst, size_t* dst_size) {
	tracestate_write_data(&hindsight_tls, &hindsight.mgr, 
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