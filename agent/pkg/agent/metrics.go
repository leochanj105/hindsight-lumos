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

	queue_totals QueueStats
	queue_ids    []int
	queues       []QueueStats

	diagnostics *Diagnostics
}

type QueueStats struct {
	trigger_count                 int
	reported_buffers              int
	evicted_buffers               int
	queue_throughput              float64
	reported_buffer_throughput    float64
	reported_buffer_throughput_mb float64
	evicted_buffer_throughput     float64
	evicted_buffer_throughput_mb  float64
	eviction_percent              float64

	diagnostics *QueueDiagnostics
}

func (stats *QueueStats) add(other *QueueStats) {
	stats.trigger_count += other.trigger_count
	stats.reported_buffers += other.reported_buffers
	stats.queue_throughput += other.queue_throughput
	stats.reported_buffer_throughput += other.reported_buffer_throughput
	stats.reported_buffer_throughput_mb += other.reported_buffer_throughput_mb
	stats.evicted_buffer_throughput += other.evicted_buffer_throughput
	stats.evicted_buffer_throughput_mb += other.evicted_buffer_throughput_mb
	stats.eviction_percent = 100 * stats.evicted_buffer_throughput / (stats.reported_buffer_throughput + stats.evicted_buffer_throughput)

	if stats.diagnostics != nil {
		stats.diagnostics.add(other.diagnostics)
	}
}

func (s *QueueStats) Str() string {
	var b strings.Builder
	fmt.Fprintf(&b, " %.1f trigs/s ", s.queue_throughput)
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
	fmt.Fprintf(&b, "  -- Triggers %v\n", s.queue_totals.Str())

	for i, trigger_id := range s.queue_ids {
		ts := s.queues[i]
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
	stats.buffer_throughput_mb = (stats.buffer_throughput * float64(agent.tm.buffer_size)) / (1024 * 1024)
	if metrics.complete_batches > 0 {
		stats.mean_batchsize = float64(metrics.complete_buffers) / float64(metrics.complete_batches)
	}
	stats.event_horizon = metrics.event_horizon

	if debug {
		diagnostics := agent.calculateDiagnostics()
		stats.diagnostics = &diagnostics
		stats.queue_totals.diagnostics = &QueueDiagnostics{}
	}

	/* Get and sort queue ids */
	for queue_id := range agent.tm.queues {
		stats.queue_ids = append(stats.queue_ids, queue_id)
	}
	sort.Ints(stats.queue_ids)

	/* Calculate stats for each trigger, plus totals */
	for _, queue_id := range stats.queue_ids {
		queue := agent.tm.queues[queue_id]
		queue_stats := agent.calculateQueueStats(duration_nanos, queue)

		if debug {
			queue_diagnostics := agent.calculateQueueDiagnostics(queue)
			queue_stats.diagnostics = &queue_diagnostics
		}

		stats.queues = append(stats.queues, queue_stats)
		stats.queue_totals.add(&queue_stats)
	}

	return stats
}

func (agent *Agent) calculateQueueStats(duration_nanos float64, queue *ManagedQueue) QueueStats {
	/* Get and reset the trigger's metrics */
	metrics := queue.queue.metrics
	queue.queue.metrics = TriggerMetrics{}

	/* Calculate stats */
	var stats QueueStats
	stats.trigger_count = metrics.count
	stats.reported_buffers = metrics.reported_buffers
	stats.queue_throughput = float64(metrics.count*1000000000) / duration_nanos
	stats.reported_buffer_throughput = float64(metrics.reported_buffers*1000000000) / duration_nanos
	stats.reported_buffer_throughput_mb = (stats.reported_buffer_throughput * float64(agent.tm.buffer_size)) / (1024 * 1024)
	stats.evicted_buffer_throughput = float64(metrics.evicted_buffers*1000000000) / duration_nanos
	stats.evicted_buffer_throughput_mb = (stats.evicted_buffer_throughput * float64(agent.tm.buffer_size)) / (1024 * 1024)
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

type QueueDiagnostics struct {
	buffers         int
	buffers_percent float64
	pending_reports int
	lru_size        int
	lru_percent     float64
}

func (d *QueueDiagnostics) Str() string {
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

func (d *QueueDiagnostics) add(other *QueueDiagnostics) {
	d.buffers += other.buffers
	d.buffers_percent += other.buffers_percent
	d.pending_reports += other.pending_reports
	d.lru_size += other.lru_size
	d.lru_percent += other.lru_percent
}

func (agent *Agent) calculateQueueDiagnostics(queue *ManagedQueue) QueueDiagnostics {
	var d QueueDiagnostics
	d.buffers = queue.queue.buffer_count
	d.buffers_percent = 100 * float64(d.buffers) / float64(agent.triggered_capacity)
	d.pending_reports = queue.queue.reporting.Size()
	d.lru_size = queue.queue.idle.Len()
	d.lru_percent = 0.001 // Deprecated
	return d
}

func (agent *Agent) calculateDiagnostics() Diagnostics {
	var d Diagnostics
	d.cache_size = agent.dm.buffer_count
	d.cache_percent = 100 * float64(d.cache_size) / float64(agent.cache_capacity)
	d.triggered_size = agent.dm.triggered.buffer_count
	d.triggered_percent = 100 * float64(d.triggered_size) / float64(agent.triggered_capacity)
	d.lru_size = agent.dm.untriggered.lru.Len()
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
