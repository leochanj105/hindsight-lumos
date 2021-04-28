package main

import (
	"fmt"
	"context"
	"github.com/geraldleizhang/hindsight/agent/pkg/agent"
    "github.com/emirpasic/gods/sets/treeset"
)

func main() {

	var set *treeset.Set
	set = treeset.NewWithIntComparator()
	set.Add(1)

	fname := "hs_integration_test"
	delay := 500
	// capacity := 10000  /// Cache capacity is loaded from shm

	agent := agent.InitAgent(fname, delay)
	fmt.Println(agent)

    ctx, _ := context.WithCancel(context.Background())
	agent.Run(ctx)

}
