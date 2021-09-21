package agent

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

type TriggerMetrics struct {
	count            int // Number of triggers
	reported_buffers int // Reporting throughput of this trigger
	evicted_buffers  int // Buffers that should have been reported but were evicted
}

type AgentMetrics struct {
	complete_batches int
	complete_buffers int
	event_horizon    time.Duration
}

type Stats struct {
	complete_batches     int
	complete_buffers     int
	buffer_throughput    float64
	buffer_throughput_mb float64
	mean_batchsize       float64
	event_horizon        time.Duration

	trigger_totals TriggerStats
	trigger_ids    []int
	triggers       []TriggerStats

	diagnostics *Diagnostics
}

type TriggerStats struct {
	trigger_count                 int
	reported_buffers              int
	evicted_buffers               int
	trigger_throughput            float64
	reported_buffer_throughput    float64
	reported_buffer_throughput_mb float64
	evicted_buffer_throughput     float64
	evicted_buffer_throughput_mb  float64
	eviction_percent              float64

	diagnostics *TriggerDiagnostics
}

func (stats *TriggerStats) add(other *TriggerStats) {
	stats.trigger_count += other.trigger_count
	stats.reported_buffers += other.reported_buffers
	stats.trigger_throughput += other.trigger_throughput
	stats.reported_buffer_throughput += other.reported_buffer_throughput
	stats.reported_buffer_throughput_mb += other.reported_buffer_throughput_mb
	stats.evicted_buffer_throughput += other.evicted_buffer_throughput
	stats.evicted_buffer_throughput_mb += other.evicted_buffer_throughput_mb
	stats.eviction_percent = 100 * stats.evicted_buffer_throughput / (stats.reported_buffer_throughput + stats.evicted_buffer_throughput)

	if stats.diagnostics != nil {
		stats.diagnostics.add(other.diagnostics)
	}
}

func (s *TriggerStats) Str() string {
	var b strings.Builder
	fmt.Fprintf(&b, " %.1f trigs/s ", s.trigger_throughput)
	fmt.Fprintf(&b, " %.1f MB/s ", s.reported_buffer_throughput_mb)
	fmt.Fprintf(&b, "(%.0f bufs/s, %d total) ", s.reported_buffer_throughput, s.reported_buffers)
	fmt.Fprintf(&b, "%.0f%% loss (%.1f MB/s)", s.eviction_percent, s.evicted_buffer_throughput_mb)
	if s.diagnostics != nil {
		fmt.Fprintf(&b, "  ||   %v", s.diagnostics.Str())
	}
	return b.String()
}

func (s *Stats) Str() string {
	var b strings.Builder
	fmt.Fprintf(&b, "EH: %d ms ", s.event_horizon/time.Millisecond)
	fmt.Fprintf(&b, "%.3f MB/s ", s.buffer_throughput_mb)
	fmt.Fprintf(&b, "(%.0f bufs/s, %d bufs total), ", s.buffer_throughput, s.complete_buffers)
	fmt.Fprintf(&b, "Avg batch %.1f; ", s.mean_batchsize)
	if s.diagnostics != nil {
		fmt.Fprintf(&b, "  ||  %v", s.diagnostics.Str())
	}
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "  -- Triggers %v\n", s.trigger_totals.Str())

	for i, trigger_id := range s.trigger_ids {
		ts := s.triggers[i]
		fmt.Fprintf(&b, "            %d - ", trigger_id)
		fmt.Fprintf(&b, "%v\n", ts.Str())
	}

	return b.String()
}

/* Calculates agent stats and resets for next iteration */
func (agent *Agent) calculateAgentStats(duration_nanos float64, debug bool) Stats {
	/* Get and reset the agent's metrics */
	metrics := agent.metrics
	agent.metrics = AgentMetrics{}

	/* Calculate stats */
	var stats Stats
	stats.complete_batches = metrics.complete_batches
	stats.complete_buffers = metrics.complete_buffers
	stats.buffer_throughput = float64(uint64(metrics.complete_buffers)*1000000000) / duration_nanos
	stats.buffer_throughput_mb = (stats.buffer_throughput * float64(agent.buffer_size)) / (1024 * 1024)
	if metrics.complete_batches > 0 {
		stats.mean_batchsize = float64(metrics.complete_buffers) / float64(metrics.complete_batches)
	}
	stats.event_horizon = metrics.event_horizon

	if debug {
		diagnostics := agent.calculateDiagnostics()
		stats.diagnostics = &diagnostics
		stats.trigger_totals.diagnostics = &TriggerDiagnostics{}
	}

	/* Get and sort trigger ids */
	for trigger_id := range agent.triggered {
		stats.trigger_ids = append(stats.trigger_ids, trigger_id)
	}
	sort.Ints(stats.trigger_ids)

	/* Calculate stats for each trigger, plus totals */
	for _, trigger_id := range stats.trigger_ids {
		trigger := agent.triggered[trigger_id]
		trigger_stats := agent.calculateTriggerStats(duration_nanos, trigger)

		if debug {
			trigger_diagnostics := agent.calculateTriggerDiagnostics(trigger)
			trigger_stats.diagnostics = &trigger_diagnostics
		}

		stats.triggers = append(stats.triggers, trigger_stats)
		stats.trigger_totals.add(&trigger_stats)
	}

	return stats
}

func (agent *Agent) calculateTriggerStats(duration_nanos float64, trigger *TriggerState) TriggerStats {
	/* Get and reset the trigger's metrics */
	metrics := trigger.metrics
	trigger.metrics = TriggerMetrics{}

	/* Calculate stats */
	var stats TriggerStats
	stats.trigger_count = metrics.count
	stats.reported_buffers = metrics.reported_buffers
	stats.trigger_throughput = float64(metrics.count*1000000000) / duration_nanos
	stats.reported_buffer_throughput = float64(metrics.reported_buffers*1000000000) / duration_nanos
	stats.reported_buffer_throughput_mb = (stats.reported_buffer_throughput * float64(agent.buffer_size)) / (1024 * 1024)
	stats.evicted_buffer_throughput = float64(metrics.evicted_buffers*1000000000) / duration_nanos
	stats.evicted_buffer_throughput_mb = (stats.evicted_buffer_throughput * float64(agent.buffer_size)) / (1024 * 1024)
	stats.eviction_percent = 100 * stats.evicted_buffer_throughput / (stats.reported_buffer_throughput + stats.evicted_buffer_throughput)

	return stats
}

type Diagnostics struct {
	cache_size        int
	cache_percent     float64
	triggered_size    int
	triggered_percent float64
	lru_size          int
}

type TriggerDiagnostics struct {
	buffers         int
	buffers_percent float64
	pending_reports int
	lru_size        int
	lru_percent     float64
}

func (d *TriggerDiagnostics) Str() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d buffers (%.0f%%) ", d.buffers, d.buffers_percent)
	fmt.Fprintf(&b, "%d pending, ", d.pending_reports)
	fmt.Fprintf(&b, "lru %.0f%% (%d)", d.lru_percent, d.lru_size)
	return b.String()
}

func (d *Diagnostics) Str() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Cache %.0f%% (%d) ", d.cache_percent, d.cache_size)
	fmt.Fprintf(&b, "Triggered %.0f%% (%d) ", d.triggered_percent, d.triggered_size)
	fmt.Fprintf(&b, "LRU %d ", d.lru_size)
	return b.String()
}

func (d *TriggerDiagnostics) add(other *TriggerDiagnostics) {
	d.buffers += other.buffers
	d.buffers_percent += other.buffers_percent
	d.pending_reports += other.pending_reports
	d.lru_size += other.lru_size
	d.lru_percent += other.lru_percent
}

func (agent *Agent) calculateTriggerDiagnostics(trigger *TriggerState) TriggerDiagnostics {
	var d TriggerDiagnostics
	d.buffers = trigger.size
	d.buffers_percent = 100 * float64(d.buffers) / float64(agent.triggered_capacity)
	d.pending_reports = trigger.unreported.Size()
	d.lru_size = trigger.lru.Len()
	d.lru_percent = 100 * float64(d.lru_size) / float64(agent.triggered_lru_capacity)
	return d
}

func (agent *Agent) calculateDiagnostics() Diagnostics {
	var d Diagnostics
	d.cache_size = agent.cache_size
	d.cache_percent = 100 * float64(d.cache_size) / float64(agent.cache_capacity)
	d.triggered_size = agent.triggered_size
	d.triggered_percent = 100 * float64(d.triggered_size) / float64(agent.triggered_capacity)
	d.lru_size = agent.lru.Len()
	return d
}

func (agent *Agent) printLoop(ctx context.Context) {
	print_every := time.Duration(1000 * time.Millisecond)
	next_print := time.NewTimer(1 * time.Millisecond)
	var last_print uint64
	for {
		select {
		case <-ctx.Done():
			return
		case <-next_print.C:
			{
				next_print.Reset(print_every)

				// Update timestamps
				now := uint64(time.Now().UnixNano())
				duration_nanos := float64(now - last_print)
				last_print = now

				debug := true
				stats := agent.calculateAgentStats(duration_nanos, debug)
				log.Print(stats.Str())
			}
		}
	}
}
