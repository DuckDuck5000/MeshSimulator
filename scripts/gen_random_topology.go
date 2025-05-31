// File: scripts/gen_random_topology.go
// Usage: go run scripts/gen_random_topology.go > testdata/random-20.yaml

package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	const (
		numNodes = 20
		pEdge    = 0.2
	)
	rand.Seed(time.Now().UnixNano())

	type NodeEntry struct {
		ID        string
		Neighbors []string
	}

	var entries []NodeEntry
	for i := 1; i <= numNodes; i++ {
		entries = append(entries, NodeEntry{
			ID:        fmt.Sprintf("Node%d", i),
			Neighbors: []string{},
		})
	}

	for i := 0; i < numNodes; i++ {
		for j := i + 1; j < numNodes; j++ {
			if rand.Float64() < pEdge {
				entries[i].Neighbors = append(entries[i].Neighbors, entries[j].ID)
				entries[j].Neighbors = append(entries[j].Neighbors, entries[i].ID)
			}
		}
	}

	fmt.Println("nodes:")
	for _, e := range entries {
		fmt.Printf("  - id: \"%s\"\n", e.ID)
		fmt.Printf("    neighbors: [")
		for idx, nb := range e.Neighbors {
			if idx > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("\"%s\"", nb)
		}
		fmt.Println("]")
	}
}
