// File: internal/ws/hub.go

package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Event represents a simulation event that will be sent to browser clients.
type Event struct {
	Type      string                 `json:"type"`      // e.g., "received", "forwarded", "dropped_network", "decrypted", etc.
	From      string                 `json:"from"`      // node ID that sent or forwarded
	To        string                 `json:"to"`        // node ID that is the recipient (or next hop)
	TTL       int                    `json:"ttl"`       // remaining TTL when this event occurred
	Payload   map[string]interface{} `json:"payload"`   // optional extra data, like "plaintext"
	Timestamp int64                  `json:"timestamp"` // UNIX milliseconds
}

// client represents one connected WebSocket client.
type client struct {
	conn *websocket.Conn
	send chan Event
}

// Hub maintains all connected WebSocket clients and broadcasts Events to them.
type Hub struct {
	// Registered clients.
	clients map[*client]bool

	// Inbound Events from the simulation that should be broadcast to all clients.
	broadcast chan Event

	// Register requests from new clients.
	register chan *client

	// Unregister requests (when a client disconnects).
	unregister chan *client

	// upgrader is used to upgrade HTTP connections to WebSocket.
	upgrader websocket.Upgrader
}

// NewHub creates a new Hub instance and initializes all channels.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *client),
		unregister: make(chan *client),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// In a production setting, you should check the origin. For now, accept any.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// Run starts the main loop for the Hub, listening for register/unregister/broadcast.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}

		case ev := <-h.broadcast:
			// Broadcast this Event to every connected client.
			for c := range h.clients {
				select {
				case c.send <- ev:
					// Successfully enqueued.
				default:
					// If the client's send channel is full or closed, drop the client.
					close(c.send)
					delete(h.clients, c)
				}
			}
		}
	}
}
func (h *Hub) BroadcastEvent(ev Event) {
	h.broadcast <- ev
}

// ServeWS upgrades the HTTP connection to a WebSocket and registers the client.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	client := &client{
		conn: conn,
		send: make(chan Event, 256),
	}
	h.register <- client

	// Start goroutines for this client.
	go client.readPump(h)
	go client.writePump()
}

// readPump reads (and discards) any messages from the client. On error, unregister.
func (c *client) readPump(h *Hub) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
		// We ignore any incoming messages from the client.
	}
}

// writePump writes broadcasted Events to the client’s WebSocket connection.
func (c *client) writePump() {
	defer c.conn.Close()
	for ev := range c.send {
		if err := c.conn.WriteJSON(ev); err != nil {
			break
		}
	}
}
