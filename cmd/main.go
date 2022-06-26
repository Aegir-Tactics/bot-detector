package main

import (
	"context"
	"fmt"
	"log"

	toolkit "github.com/aegir-tactics/bot-detector"
)

func main() {
	e, err := toolkit.NewEngine()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	fmt.Println("starting")

	if err := e.TopAddresses(ctx, toolkit.Addresses); err != nil {
		log.Fatal(err)
	}
	fmt.Println("TYPE: ITERATION: DEPTH: ADDRESS")

	var tree int
	for _, node := range e.Trees {
		if node.Parent != nil {
			continue
		}
		tree++

		var star string
		if _, ok := toolkit.Addresses[node.Address]; ok {
			star = "*"
		}
		fmt.Printf("TRUNK: %v, 0: %s%s\n", tree, star, node.Address)
		PrintKids(tree, node.Children, 1)
	}
}

func PrintKids(tree int, kids []*toolkit.Node, lvl int) {
	if len(kids) == 0 {
		return
	}

	for _, child := range kids {
		var star string
		if _, ok := toolkit.Addresses[child.Address]; ok {
			star = "*"
		}
		fmt.Printf("CHILD: %v, %v: %s%s\n", tree, lvl, star, child.Address)
		PrintKids(tree, child.Children, lvl+1)
	}
}
