package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	. "cache"
	. "memory"
	. "queue"
	. "server"
)

func hindsight_init() {
	isConfig := conf_init()
	if !isConfig {
		fmt.Println("Failed to load config file")
		return
	}

	SharedPool.Pool = MemInit("/dev/shm/pool", Cap*Buf_length*4)
	SharedDict.Dict = MemInit("/dev/shm/dict", 3200)
	Complete = QueueInit("/dev/shm/complete_queue", Cap)
	Available = QueueInit("/dev/shm/available_queue", Cap)
	Triggers = QueueInit("/dev/shm/triggers_queue", Cap)

	CacheInit(Cap)
	AvailableInit(Cap)
	// QueueInitTest()
}

func conf_init() bool {
	conf_file, _ := os.Open("../conf/default.conf")
	// if err != nil {
	// 	conf_file, err = os.Open("/etc/hindsight.conf")
	// 	if err != nil {
	// 		fmt.Println("Please check conf file")
	// 		return false
	// 	}
	// }
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
			Server_port, _ = strconv.Atoi(strings.Split(scanner.Text(), " ")[1])
		}
		if strings.Contains(scanner.Text(), "lc_addr") {
			LC_addr = strings.Split(scanner.Text(), " ")[1]
		}
		if strings.Contains(scanner.Text(), "lc_port") {
			LC_port, _ = strconv.Atoi(strings.Split(scanner.Text(), " ")[1])
		}
	}

	if Server_addr == "" || Server_port == 0 {
		fmt.Println("Please declare agent addr and port")
		return false
	}

	return true
}

func run() {
	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		RunQueueServer()
		wg.Done()
	}()

	go func() {
		RunTriggerServer()
		wg.Done()
	}()

	wg.Wait()
}

func stat() {
	PrintQueueStat()
}

func main() {
	hindsight_init()

	run()
	// stat()

	// queue := QueueInit("/dev/shm/queue_test", 200)

	// for {
	// 	data := QueueGet(queue)
	// 	fmt.Println("get data", data)
	// }

	// QueueTest(queue)
	// mem := MemInit("/dev/shm/pool_test", 1008)

	// fmt.Println(mem[0:8], BytesToInt64(mem[0:8]))
	// fmt.Println(mem[8:12], BytesToInt32(mem[8:12]))
	// for i := 0; i < 250; i++ {
	// 	fmt.Println(mem[12+i*4:16+i*4], BytesToInt32(mem[12+i*4:16+i*4]))
	// }

}
