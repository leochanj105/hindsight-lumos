package main

import (
	"fmt"
	"github.com/geraldleizhang/hindsight/agent/pkg/agent"
    "github.com/emirpasic/gods/sets/treeset"
)

func main() {

	var set *treeset.Set
	set = treeset.NewWithIntComparator()
	set.Add(1)

	fname := "blah"
	delay := 500
	capacity := 10000

	agent := agent.InitAgent(fname, delay, capacity)
	fmt.Println(agent)

	agent.Run()

}
