// File: internal/node/node.go

package node

import (
	"time"

	"meshsim/internal/crypto"
	"meshsim/internal/ws"
)

// Message represents an encrypted packet traveling through the mesh.
type Message struct {
	From      string    // sender ID
	To        string    // final recipient ID
	TTL       int       // remaining hops
	Timestamp time.Time // original send time

	Cipher []byte   // nacl “box” ciphertext
	Nonce  [24]byte // nonce used for encryption
}

// Node is a single participant in the mesh network.
type Node struct {
	ID         string
	Neighbors  []string
	Inbound    chan Message
	Outbound   chan Message
	PublicKeys map[string][]byte // NodeID → public key bytes
	PrivateKey *[32]byte         // this node’s private key
	hub        *ws.Hub           // WebSocket hub to broadcast events
}

// Stop gracefully shuts down the node's goroutines.
func (n *Node) Stop() {
	close(n.Inbound)
}

// NewNode creates a new Node instance.
func NewNode(
	id string,
	neighbors []string,
	pubKeys map[string][]byte,
	privKey *[32]byte,
	hub *ws.Hub,
) *Node {
	return &Node{
		ID:         id,
		Neighbors:  neighbors,
		Inbound:    make(chan Message, 10),
		Outbound:   make(chan Message, 10),
		PublicKeys: pubKeys,
		PrivateKey: privKey,
		hub:        hub,
	}
}

// Start begins the node’s main loop: listening for inbound messages, decrypting or forwarding.
func (n *Node) Start() {
	go func() {
		defer close(n.Outbound)
		for msg := range n.Inbound {
			// 1) Broadcast “received” event (before TTL check)
			n.hub.BroadcastEvent(ws.Event{
				Type:      "received",
				From:      msg.From,
				To:        n.ID,
				TTL:       msg.TTL,
				Payload:   map[string]interface{}{},
				Timestamp: time.Now().UnixMilli(),
			})

			// 2) Drop if TTL expired
			if msg.TTL <= 0 {
				n.hub.BroadcastEvent(ws.Event{
					Type:      "dropped_ttl",
					From:      msg.From,
					To:        n.ID,
					TTL:       msg.TTL,
					Payload:   map[string]interface{}{},
					Timestamp: time.Now().UnixMilli(),
				})
				continue
			}

			// 3) If this node is the final recipient, decrypt & broadcast “decrypted”
			if msg.To == n.ID {
				senderPubBytes := n.PublicKeys[msg.From]
				var senderPub [32]byte
				copy(senderPub[:], senderPubBytes)

				plaintext, err := crypto.DecryptMessage(
					msg.Cipher,
					&msg.Nonce,
					&senderPub,
					n.PrivateKey,
				)
				if err != nil {
					// broadcast decryption failure
					n.hub.BroadcastEvent(ws.Event{
						Type:      "decrypt_failed",
						From:      msg.From,
						To:        n.ID,
						TTL:       msg.TTL,
						Payload:   map[string]interface{}{"error": err.Error()},
						Timestamp: time.Now().UnixMilli(),
					})
				} else {
					// broadcast decrypted content
					n.hub.BroadcastEvent(ws.Event{
						Type:      "decrypted",
						From:      msg.From,
						To:        n.ID,
						TTL:       msg.TTL,
						Payload:   map[string]interface{}{"plaintext": string(plaintext)},
						Timestamp: time.Now().UnixMilli(),
					})
				}
				continue
			}

			// 4) Otherwise, forward to neighbors (decrement TTL) and broadcast “forwarded”
			msg.TTL--
			n.hub.BroadcastEvent(ws.Event{
				Type:      "forwarded",
				From:      n.ID,
				To:        "", // blank because it will be sent to all neighbors
				TTL:       msg.TTL,
				Payload:   map[string]interface{}{},
				Timestamp: time.Now().UnixMilli(),
			})
			n.Outbound <- msg
		}
	}()
}
