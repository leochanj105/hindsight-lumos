package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/geraldleizhang/hindsight/agent/pkg/agent"
	"github.com/geraldleizhang/hindsight/agent/pkg/collector_new"
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

func main() {

	isLC := flag.Bool("lc", false, "Log Collector")
	serv_temp := flag.String("serv", "", "Service name")
	// isReport := flag.Bool("report", true, "If report to LC (or local mode)")
	delayf := flag.Int("delay", 0, "Delayed trigger time")
	ratelimit := flag.Float64("rate", 0, "Rate limit for reporting traces in MB/s")

	per_trigger_limits := make(triggerRateLimitFlags)
	flag.Var(&per_trigger_limits, "l", "A per-trigger rate limit in the form trigger_id,rate where trigger_id is an integer and rate is a float")

	flag.Parse()

	service_name := *serv_temp
	delay := uint64(1000000 * (*delayf))

	isConfig := util.Conf_init(service_name)
	if !isConfig {
		fmt.Println("Failed to load config file")
		return
	}

	ctx, _ := context.WithCancel(context.Background())

	if *isLC == true {
		fmt.Println("running lc")
		lc := collector_new.InitLC()
		lc.Run(ctx)
	} else {
		fmt.Println("running server")
		agent := agent.InitAgent2(service_name, delay, *ratelimit, per_trigger_limits)
		agent.Run(ctx)
	}

}
