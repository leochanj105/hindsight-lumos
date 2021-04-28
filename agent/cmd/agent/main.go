package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	. "github.com/geraldleizhang/hindsight/agent/pkg/agent"
	. "github.com/geraldleizhang/hindsight/agent/pkg/cache"
	. "github.com/geraldleizhang/hindsight/agent/pkg/collector"
	. "github.com/geraldleizhang/hindsight/agent/pkg/memory"
	. "github.com/geraldleizhang/hindsight/agent/pkg/server"
	. "github.com/geraldleizhang/hindsight/agent/pkg/util"
)

var service_name string

func hindsight_init() {
	SharedPool.Pool = MemInit("/dev/shm/pool_"+service_name, Cap*Buf_length*4)
	SharedDict.Dict = MemInit("/dev/shm/dict_"+service_name, 3200)
	Complete = QueueInit("/dev/shm/complete_queue_"+service_name, Cap)
	Available = QueueInit("/dev/shm/available_queue_"+service_name, Cap)
	Triggers = QueueInit("/dev/shm/triggers_queue_"+service_name, Cap)

	CacheInit(Cap)
	AvailableInit(Cap)
	// QueueInitTest()
}

func run(isReport bool) {
	ServerInit()
	if isReport {
		wg := new(sync.WaitGroup)
		wg.Add(4)

		go func() {
			RunQueueServer()
			wg.Done()
		}()

		go func() {
			RunTriggerServer()
			wg.Done()
		}()

		go func() {
			RunResponseServer()
			wg.Done()
		}()

		go func() {
			RunAgent()
			wg.Done()
		}()

		wg.Wait()
	} else {
		RunQueueServer()
	}

}

func run_lc() {
	CollectorInit()
	wg := new(sync.WaitGroup)
	wg.Add(3)

	go func() {
		RunLCResponseServer()
		wg.Done()
	}()

	go func() {
		RunRetrievalHandler()
		wg.Done()
	}()

	go func() {
		RunCollector()
		wg.Done()
	}()

	wg.Wait()
}

func stat() {
	PrintQueueStat()
}

func main() {
	DEBUG = 0

	agent := InitAgent("blah", 500)

	isLC := flag.Bool("lc", false, "Log Collector")
	serv_temp := flag.String("serv", "", "Service name")
	isReport := flag.Bool("report", true, "If report to LC (or local mode)")
	delay := flag.Int("delay", 0, "Delayed trigger time")

	flag.Parse()
	service_name = *serv_temp
	Delay = int64(1000000 * (*delay))

	isConfig := conf_init()
	if !isConfig {
		fmt.Println("Failed to load config file")
		return
	}

	if *isLC == true {
		fmt.Println("running lc")
		run_lc()
	} else {
		fmt.Println("running server")
		hindsight_init()
		run(*isReport)
	}
}
