package agent

/* Interface for generating telemetry.  Hindsight's
agent, coordinator, and collector all implement this interface
to provide telemetry data */
type TelemetryGenerator interface {
	Headers() []string
	NextData() []map[string]string
}
