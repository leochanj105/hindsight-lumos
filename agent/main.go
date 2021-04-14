package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	. "cache"
	. "collector"
	. "memory"
	. "queue"
	. "server"
	. "util"
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

func conf_init() bool {
	conf_file, err := os.Open("/etc/hindsight_conf/" + service_name + ".conf")
	if err != nil {
		conf_file, err = os.Open("/etc/hindsight_conf/default.conf")
		if err != nil {
			fmt.Println("Please check conf file")
			return false
		}
		// fmt.Println("Please check conf file", "/etc/hindsight_conf/"+service_name+".conf")
		// return false
	}
	defer conf_file.Close()

	// Service_name = serv_name

	scanner := bufio.NewScanner(conf_file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "cap") {
			Cap, _ = strconv.Atoi(strings.Split(scanner.Text(), " ")[1])
		}
		if strings.Contains(scanner.Text(), "buf_length") {
			Buf_length, _ = strconv.Atoi(strings.Split(scanner.Text(), " ")[1])
		}
		if strings.Contains(scanner.Text(), "addr") && !strings.Contains(scanner.Text(), "lc") {
			Server_addr = strings.Split(scanner.Text(), " ")[1]
		}
		if strings.Contains(scanner.Text(), "port") && !strings.Contains(scanner.Text(), "lc") {
			Server_port = strings.Split(scanner.Text(), " ")[1]
		}
		if strings.Contains(scanner.Text(), "lc_addr") {
			LC_addr = strings.Split(scanner.Text(), " ")[1]
		}
		if strings.Contains(scanner.Text(), "lc_port") {
			LC_port = strings.Split(scanner.Text(), " ")[1]
		}
	}

	fmt.Println("config file loaded, cap=", Cap, "addr=", Server_addr, "port=", Server_port)

	if Server_addr == "" || Server_port == "0" {
		fmt.Println("Please declare agent addr and port")
		return false
	}

	return true
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

	isLC := flag.Bool("lc", false, "Log Collector")
	serv_temp := flag.String("serv", "", "Service name")
	isReport := flag.Bool("report", true, "If report to LC (or local mode)")

	flag.Parse()
	service_name = *serv_temp

	if *isLC == true {
		fmt.Println("running lc")
		run_lc()
	} else {
		fmt.Println("running server")
		isConfig := conf_init()
		if !isConfig {
			fmt.Println("Failed to load config file")
			return
		}
		hindsight_init()
		run(*isReport)
	}
}
