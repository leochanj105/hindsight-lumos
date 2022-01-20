package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/geraldleizhang/hindsight/agent/pkg/coordinator"
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

// TODO different main methods for different cmds..........
func main() {

	port := flag.String("port", "5252", "Port that the coordinator listens on for connections from agents")

	flag.Parse()

	ctx, _ := context.WithCancel(context.Background())

	// // Not sure if needed
	// util.Conf_init("lc")

	var c coordinator.CoordinatorServer
	c.Init(*port)
	c.Run(ctx)
}
