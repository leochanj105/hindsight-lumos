package main

// IMPORTANT: for the below to work, must do:
//   export CGO_LDFLAGS_ALLOW=".*"

/*
#cgo CFLAGS: -I${SRCDIR}/../../../client/src
#cgo LDFLAGS: ${SRCDIR}/../../../client/lib/libtracer.a

#include "agentapi.h"

*/
import "C"

import (
    "fmt"
    "time"
    "github.com/geraldleizhang/hindsight/agent/pkg/memory"
)



func drain_forever(api *memory.AgentAPI) {
    last_print := int(time.Now().UnixNano())
    print_every := 1000000000
    count := 0
    sum := 0

    for {
        now := int(time.Now().UnixNano())
        if ((now - last_print) > print_every) {
            tput := (sum * print_every) / (now - last_print)
            batchsize := float32(sum) / float32(count)
            fmt.Println("Throughput:", tput, "Average batch:", batchsize)
            last_print = now
            count = 0
            sum = 0
        }

        var buffers []memory.CompleteBuffer
        max_backoff := 100000
        backoff := int(10)
        for {
            buffers = api.GetComplete()
            if len(buffers) > 0 {
                break
            }

            time.Sleep(time.Duration(backoff) * time.Nanosecond)
            backoff *= 2
            if (backoff > max_backoff) {
                backoff = max_backoff
            }
        }

        count += 1
        sum += len(buffers)

        ids := make([]int, len(buffers))
        for i, buf := range buffers {
            ids[i] = buf.Buffer_id
        }

        api.PutAvailable(ids)
    }
}

func main() {
    fmt.Println("Hello world!")

    fname := "hs_integration_test"

    // f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0666)
    // if err != nil {
    //  fmt.Println("open file failed:", err)
    // }
    // fd := int(f.Fd())
    // fmt.Println("opened ", fd)

    // fi, err := f.Stat()
    // if err != nil {
    //  fmt.Println("stat failed:",err)
    // }
    // fmt.Println("size is ", fi.Size())

    // p, err := syscall.Mmap(fd, 0, int(fi.Size()), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)

    // fmt.Printf("%T\n", p)

    agentapi := memory.InitAgentAPI(fname)
    drain_forever(agentapi)



    // fmt.Println(q)

    // fmt.Println(q.meta)

    // md := C.QueueMetadata(p)
}
