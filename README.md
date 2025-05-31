Secure Mesh Communication Simulator
A lightweight, browser-based 2D visualization of an encrypted mesh‐network simulation. MeshSim consists of:

Go backend (in cmd/meshsim)

Serves REST endpoints for loading topologies, listing nodes, and sending messages.

Maintains an in‐memory simulation of nodes, each with a NaCl keypair.

Uses Gorilla Mux for routing and Gorilla WebSocket for real‐time event streaming.

Front-end UI (in web/static)

A single HTML page (index.html) with embedded CSS layouts.

A D3.js‐powered SVG for force‐directed graph visualization (app.js).

Controls for selecting existing topologies, creating custom topologies (paste YAML or “build via form”), adjusting drop rate & TTL, and sending test messages.

When you start the server, you can open http://localhost:8080/ in your browser. The left pane contains the controls (topology selection, drop‐rate slider, TTL slider, sender/recipient, etc.). The right pane is split vertically into:

Visualization: A 2/3‐height D3 force‐directed graph of the mesh.

Event Log: A 1/3‐height scrolling log of real‐time events (forward, drop, decrypted, delivered).

.
├── cmd
│   └── meshsim
│       ├── main.go           ← Go server entrypoint
│       └── go.mod / go.sum
├── internal
│   ├── crypto               ← NaCl box keypair generation & Encrypt/Decrypt functions
│   ├── network              ← Mesh‐network logic (node registration, message routing, drop simulation)
│   ├── node                 ← Node‐level goroutines, WebSocket event broadcasts
│   └── ws                   ← Gorilla WebSocket Hub, event struct definitions
├── pkg
│   └── config               ← YAML loading & Topology struct
├── testdata
│   ├── clustered-12.yaml
│   ├── fullmesh-10.yaml
│   ├── linear-20.yaml
│   └── ...                  ← Any other sample topology files
├── web
│   └── static
│       ├── index.html       ← Single‐page UI (controls + SVG + log)
│       └── app.js           ← Front‐end logic (D3, WebSocket, AJAX)
├── README.md                ← (This file)
└── go.mod / go.sum          ← Module declarations at project root


Prerequisites
Go 1.24+ (any recent Go 1.x should work)

Node.js / npm (optional, only if you want to install front‐end dependencies manually; not strictly required here)

A modern browser (Chrome, Firefox, Safari, Edge) for the front end.

All third‐party Go modules (e.g. Gorilla Mux, Gorilla WebSocket, gopkg.in/yaml.v3) are fetched automatically via go mod tidy / go mod download.


Getting Started
Clone the repository

bash
Copy
Edit
git clone https://github.com/your‐username/mesh‐sim.git
cd mesh‐sim
Ensure Go modules are up to date

bash
Copy
Edit
go mod tidy
Run the server

bash
Copy
Edit
go run cmd/meshsim/main.go
You should see:

arduino
Copy
Edit
Server is running at http://localhost:8080
→ WebSocket endpoint: ws://localhost:8080/ws
Open your browser and navigate to:

arduino
Copy
Edit
http://localhost:8080/

