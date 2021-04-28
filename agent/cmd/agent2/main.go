package main

import (
	"fmt"
	"context"
	"flag"
	"sync"
	"github.com/geraldleizhang/hindsight/agent/pkg/agent"
	"github.com/geraldleizhang/hindsight/agent/pkg/collector"
	"github.com/geraldleizhang/hindsight/agent/pkg/util"
)

func run_lc() {
	collector.CollectorInit()
	wg := new(sync.WaitGroup)
	wg.Add(3)

	go func() {
		collector.RunLCResponseServer()
		wg.Done()
	}()

	go func() {
		collector.RunRetrievalHandler()
		wg.Done()
	}()

	go func() {
		collector.RunCollector()
		wg.Done()
	}()

	wg.Wait()
}

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
		run_lc()
	} else {
		fmt.Println("running server")
		agent := agent.InitAgent(service_name, delay)
	    ctx, _ := context.WithCancel(context.Background())
		agent.Run(ctx)
	}

}
