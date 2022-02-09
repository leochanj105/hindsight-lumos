package agent

import (
	"context"
	"fmt"
	"time"
)

/* Ties generators to receivers; reports telemetry with a configurable interval */
type PeriodicReporter struct {
	interval  time.Duration
	generator TelemetryGenerator
	receiver  TelemetryReceiver
}

func (reporter *PeriodicReporter) Init(interval time.Duration, generator TelemetryGenerator, receiver TelemetryReceiver) {
	reporter.interval = interval
	reporter.generator = generator
	reporter.receiver = receiver
}

func (reporter *PeriodicReporter) Run(ctx context.Context) (err error) {
	// User must call Init before calling Run
	if reporter.generator == nil || reporter.receiver == nil {
		return fmt.Errorf("Attempted to run an uninitialized reporter")
	}
	// First write the headers
	err = reporter.receiver.Init(reporter.generator.Headers())
	if err != nil {
		return
	}

	// Write data forever
	ticker := time.NewTicker(reporter.interval)
	for {
		select {
		case <-ctx.Done():
			err = reporter.receiver.Close()
			return
		case <-ticker.C:
			err = reporter.receiver.Report(reporter.generator.NextData())
			if err != nil {
				fmt.Println("Error reporting telemetry", err)
				return
			}
		}
	}
}
