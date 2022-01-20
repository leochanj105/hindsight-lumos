package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/geraldleizhang/hindsight/agent/pkg/agent"
	"github.com/geraldleizhang/hindsight/agent/pkg/util"
)

type triggerRateLimitFlags map[int]float64

func (rates *triggerRateLimitFlags) String() string {
	var b strings.Builder
	for trigger_id, rate := range *rates {
		fmt.Fprintf(&b, "%d=%.1f ", trigger_id, rate)
	}
	return b.String()
}

func (i *triggerRateLimitFlags) Set(value string) error {
	splits := strings.Split(value, ",")
	if len(splits) != 2 {
		return fmt.Errorf("Invalid rate %v -- must be of the form int,float", value)
	}
	trigger_id, err := strconv.ParseInt(splits[0], 10, 64)
	if err != nil {
		return err
	}
	rate, err := strconv.ParseFloat(splits[1], 64)
	if err != nil {
		return err
	}

	(*i)[int(trigger_id)] = rate
	return nil
}

func resolveConfigValue(key string, value string, legacyconfigvalue string, service_name string) string {
	if value == "" {
		value = legacyconfigvalue
		fmt.Printf("  %s=%s (%s.conf)\n", key, value, service_name)
	} else {
		fmt.Printf("  %s=%s (command line)\n", key, value)
	}
	return value
}

// TODO different main methods for different cmds..........
func main() {

	serv := flag.String("serv", "", "Service name")
	hostname := flag.String("host", "", "Hostname or IP of this agent.  If not specified, uses `addr` from the legacy config file")
	port := flag.String("port", "", "Port to run the agent on.  If not specified, uses `port` from the legacy config file.")
	lc_addr := flag.String("lc", "", "Address of the log collector in form hostname:port.  If not specified, uses `lc_addr`:`lc_port` from the legacy config file.")
	r_addr := flag.String("r", "", "Address of the reporting backend in form hostname:port.  If not specified, uses `r_addr`:`r_port` from the legacy config file.")
	// isReport := flag.Bool("report", true, "If report to LC (or local mode)")
	delayf := flag.Int("delay", 0, "Used for experimental purposes.  If specified, this delays the reporting of triggers by the specified delay (in nanoseconds).  Default to 0 - no delay.")
	reportingratelimit := flag.Float64("rate", 0, "Rate limit for reporting traces in MB/s.  Set to 0 to disable.  Default 0.")
	triggerratelimit := flag.Float64("triggerrate", 0, "Rate limit for a spammy trigger in triggers/s.  Set to 0 to disable.  Default 10000.")

	per_trigger_limits := make(triggerRateLimitFlags)
	flag.Var(&per_trigger_limits, "l", "A per-trigger reporting rate limit in the form queue_id,rate where queue_id is an integer and rate is a float representing a reporting limit in MB/s.  This flag can be set multiple times to provide rate limits for different triggers.")

	flag.Parse()

	delay := uint64(1000000 * (*delayf))

	isConfig := util.Conf_init(*serv)
	if !isConfig {
		fmt.Println("Failed to load config file for", *serv)
		return
	}

	fmt.Println("Running agent", *serv)
	*hostname = resolveConfigValue("hostname", *hostname, util.Server_addr, *serv)
	*port = resolveConfigValue("port", *port, util.Server_port, *serv)
	*lc_addr = resolveConfigValue("lc_addr", *lc_addr, util.Coordinator_addr+":"+util.Coordinator_port, *serv)
	*r_addr = resolveConfigValue("r_addr", *r_addr, util.Reporting_addr+":"+util.Reporting_port, *serv)

	ctx, _ := context.WithCancel(context.Background())

	agent := agent.InitAgent2(*serv, *hostname, *port, *lc_addr, *r_addr, delay, *reportingratelimit, *triggerratelimit, per_trigger_limits)
	agent.Run(ctx)
}
