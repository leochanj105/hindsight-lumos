package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/geraldleizhang/hindsight/agent/pkg/agent"
	"github.com/geraldleizhang/hindsight/agent/pkg/collector_new"
	"github.com/geraldleizhang/hindsight/agent/pkg/util"
)

func main() {

	isLC := flag.Bool("lc", false, "Log Collector")
	serv_temp := flag.String("serv", "", "Service name")
	// isReport := flag.Bool("report", true, "If report to LC (or local mode)")
	delayf := flag.Int("delay", 0, "Delayed trigger time")
	flag.Parse()

	service_name := *serv_temp
	delay := uint64(1000000 * (*delayf))

	isConfig := util.Conf_init(service_name)
	if !isConfig {
		fmt.Println("Failed to load config file")
		return
	}

	if *isLC == true {
		fmt.Println("running lc")
		lc := collector_new.InitLC()
		ctx, _ := context.WithCancel(context.Background())
		lc.Run(ctx)
	} else {
		fmt.Println("running server")
		agent := agent.InitAgent(service_name, delay)
		ctx, _ := context.WithCancel(context.Background())
		agent.Run(ctx)
	}

}
