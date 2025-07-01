// File: cmd/meshsim/main.go

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

	"meshsim/internal/crypto"
	"meshsim/internal/network"
	"meshsim/internal/node"
	"meshsim/internal/ws"
	"meshsim/pkg/config"
)

// Globals to hold the “current” simulation state:
var (
	hub        *ws.Hub
	netSim     *network.Network
	keys       map[string]*crypto.KeyPair
	publicKeys map[string][]byte
	topo       *config.Topology
	currentTTL int
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// 1) Create and start the WebSocket Hub
	hub = ws.NewHub()
	go hub.Run()

	// 2) Create the router
	r := mux.NewRouter()

	// 3) Static file serving for frontend assets
	staticFs := http.FileServer(http.Dir("web/static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFs))
	topoFs := http.FileServer(http.Dir("testdata"))
	r.PathPrefix("/topo-files/").Handler(http.StripPrefix("/topo-files/", topoFs))

	// 4) API endpoints

	// 4a) GET /topologies → list all YAML files under testdata/
	r.HandleFunc("/topologies", handleGetTopologies).Methods("GET")

	// 4b) POST /configure → load topology, set dropRate, ttl, build mesh
	r.HandleFunc("/configure", handleConfigure).Methods("POST")

	// 4c) GET /nodes → return JSON list of node IDs in current topo
	r.HandleFunc("/nodes", handleGetNodes).Methods("GET")

	// 4d) POST /send → JSON { "from": ..., "to": ... }
	r.HandleFunc("/send", handleSend).Methods("POST")

	// 4e) GET /ws → upgrade to WebSocket for real-time events
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	})

	// 5) GET / → serve index.html
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("web", "static", "index.html"))
	})

	// 6) Start HTTP server
	addr := ":8080"
	fmt.Println("Server is running at http://localhost" + addr)
	fmt.Println("→ WebSocket endpoint: ws://localhost" + addr + "/ws")
	log.Fatal(http.ListenAndServe(addr, r))
}

// handleGetTopologies reads the testdata/ directory and returns a JSON array of all .yaml/.yml filenames.
func handleGetTopologies(w http.ResponseWriter, r *http.Request) {
	dir := "testdata"
	entries, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "cannot read testdata directory", http.StatusInternalServerError)
		return
	}
	var list []string
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			name := entry.Name()
			if filepath.Ext(name) == ".yaml" || filepath.Ext(name) == ".yml" {
				list = append(list, name)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// configureRequest is the JSON schema for POST /configure
type configureRequest struct {
	Topology   string  `json:"topology"`
	CustomYAML string  `json:"customYaml"`
	DropRate   float64 `json:"dropRate"`
	TTL        int     `json:"ttl"`
}

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	var req configureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// If a simulation is already running, shut it down before starting a new one.
	if netSim != nil {
		netSim.Shutdown()
	}

	// 1) Validate dropRate and TTL
	if req.DropRate < 0.0 || req.DropRate > 1.0 {
		http.Error(w, "dropRate must be between 0.0 and 1.0", http.StatusBadRequest)
		return
	}
	if req.TTL < 1 {
		http.Error(w, "ttl must be ≥ 1", http.StatusBadRequest)
		return
	}

	// 2) Decide whether to load from an existing file
	//    or to parse raw YAML from the payload.
	var loadedTopo *config.Topology
	if strings.TrimSpace(req.CustomYAML) != "" {
		// --- CUSTOM YAML PATH ---
		// Parse the raw YAML string from `req.CustomYAML`.
		var topoFromString config.Topology
		if err := yaml.Unmarshal([]byte(req.CustomYAML), &topoFromString); err != nil {
			http.Error(w, "failed to parse custom YAML: "+err.Error(), http.StatusBadRequest)
			return
		}
		loadedTopo = &topoFromString

	} else {
		// --- EXISTING FILE PATH ---
		if strings.TrimSpace(req.Topology) == "" {
			http.Error(w, "no topology file specified", http.StatusBadRequest)
			return
		}
		// Verify the file exists in testdata/
		yamlPath := filepath.Join("testdata", req.Topology)
		if _, err := os.Stat(yamlPath); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "topology file not found", http.StatusBadRequest)
			} else {
				http.Error(w, "error accessing topology file", http.StatusInternalServerError)
			}
			return
		}
		// Use your existing LoadConfig(filePath) function
		t, err := config.LoadConfig(yamlPath)
		if err != nil {
			http.Error(w, "failed to parse topology file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		loadedTopo = t
	}

	// At this point, `loadedTopo` points to a valid *config.Topology,
	// whether we got it from a file or from the raw YAML string.

	// 3) Generate fresh keypairs & publicKeys
	keys = make(map[string]*crypto.KeyPair)
	publicKeys = make(map[string][]byte)
	for _, nodeCfg := range loadedTopo.Nodes {
		kp, err := crypto.GenerateKeyPair()
		if err != nil {
			http.Error(w, "key generation error", http.StatusInternalServerError)
			return
		}
		keys[nodeCfg.ID] = kp
		publicKeys[nodeCfg.ID] = kp.PublicKey[:]
	}

	// 4) Build a new Network
	currentTTL = req.TTL
	topo = loadedTopo
	netSim = network.NewNetwork(req.DropRate, hub)

	// 5) Instantiate nodes
	for _, nodeCfg := range topo.Nodes {
		n := node.NewNode(
			nodeCfg.ID,
			nodeCfg.Neighbors,
			publicKeys,
			keys[nodeCfg.ID].PrivateKey,
			hub,
		)
		netSim.AddNode(n)
		n.Start()
	}

	// 6) Start network forwarding/dropping
	netSim.Connect()

	// 7) Respond “OK”
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"configured"}`))
}

// handleGetNodes returns JSON array of current node IDs. If no topology is configured yet, returns [].
func handleGetNodes(w http.ResponseWriter, r *http.Request) {
	if topo == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	var ids []string
	for _, cfg := range topo.Nodes {
		ids = append(ids, cfg.ID)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ids)
}

// sendRequest is the JSON schema for POST /send
type sendRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// handleSend encrypts a test message (using currentTTL) and enqueues it into the mesh.
func handleSend(w http.ResponseWriter, r *http.Request) {
	if netSim == nil {
		http.Error(w, "simulation not configured", http.StatusBadRequest)
		return
	}

	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate sender/recipient exist
	if _, ok := netSim.Nodes[req.From]; !ok {
		http.Error(w, "invalid 'from' node ID", http.StatusBadRequest)
		return
	}
	if _, ok := netSim.Nodes[req.To]; !ok {
		http.Error(w, "invalid 'to' node ID", http.StatusBadRequest)
		return
	}

	// Encrypt the payload
	senderKP := keys[req.From]
	var recvPub [32]byte
	copy(recvPub[:], publicKeys[req.To])
	plaintext := []byte(fmt.Sprintf("Hello from %s → %s", req.From, req.To))

	ciphertext, nonce, err := crypto.EncryptMessage(plaintext, senderKP.PrivateKey, &recvPub)
	if err != nil {
		http.Error(w, "encryption failed", http.StatusInternalServerError)
		return
	}

	// Build and enqueue the message
	msg := node.Message{
		From:      req.From,
		To:        req.To,
		TTL:       currentTTL,
		Timestamp: time.Now(),
		Cipher:    ciphertext,
		Nonce:     *nonce,
	}
	netSim.Nodes[req.From].Inbound <- msg

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"message queued"}`))
}
