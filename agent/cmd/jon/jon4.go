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
    "context"
    "github.com/geraldleizhang/hindsight/agent/pkg/memory"
)



func drain_forever(api *memory.GoAgentAPI, ctx context.Context) {
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

        select {
        case breadcrumbs := <- api.Breadcrumbs:
            count += 1
            sum += len(breadcrumbs)
            for _, crumb := range breadcrumbs {
                fmt.Println("Received breadcrumb:", crumb.Request_id, " at ", crumb.Address)
            }
        }
    }
    select {
        case <- ctx.Done():
            return
    }
}

func main() {
    fmt.Println("Hello world!")

    fname := "hs_integration_test"

    agentapi := memory.InitGoAgentAPI(fname)

    ctx, _ := context.WithCancel(context.Background())

    go agentapi.Run(ctx)
    drain_forever(agentapi, ctx)

}
