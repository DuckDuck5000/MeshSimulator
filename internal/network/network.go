// File: internal/network/network.go

package network

import (
	"math/rand"
	"time"

	"meshsim/internal/node"
	"meshsim/internal/ws"
)

// Network simulates a mesh of Node instances, injecting random drops.
type Network struct {
	Nodes    map[string]*node.Node
	DropRate float64
	hub      *ws.Hub
}

// NewNetwork initializes a Network with the given drop rate and WebSocket hub.
func NewNetwork(dropRate float64, hub *ws.Hub) *Network {
	return &Network{
		Nodes:    make(map[string]*node.Node),
		DropRate: dropRate,
		hub:      hub,
	}
}

// AddNode registers a Node into this mesh.
func (net *Network) AddNode(n *node.Node) {
	net.Nodes[n.ID] = n
}

// Connect starts a goroutine for each node. For every outbound message, it fans out to each neighbor (unless dropped).
func (net *Network) Connect() {
	for _, n := range net.Nodes {
		go func(n *node.Node) {
			for msg := range n.Outbound {
				for _, neighborID := range n.Neighbors {
					target, exists := net.Nodes[neighborID]
					if !exists {
						continue
					}

					// 1) Randomly drop according to DropRate
					if rand.Float64() < net.DropRate {
						net.hub.BroadcastEvent(ws.Event{
							Type:      "dropped_network",
							From:      n.ID,
							To:        neighborID,
							TTL:       msg.TTL,
							Payload:   map[string]interface{}{},
							Timestamp: time.Now().UnixMilli(),
						})
						continue
					}

					// 2) Optional small random latency
					time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

					// 3) Actual delivery
					net.hub.BroadcastEvent(ws.Event{
						Type:      "delivered",
						From:      n.ID,
						To:        neighborID,
						TTL:       msg.TTL,
						Payload:   map[string]interface{}{},
						Timestamp: time.Now().UnixMilli(),
					})
					target.Inbound <- msg
				}
			}
		}(n)
	}
}
