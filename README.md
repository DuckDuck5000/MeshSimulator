# Secure Mesh Communication Simulator

A lightweight, browser-based 2D visualization of an encrypted mesh‐network simulation. MeshSim consists of:

## Backend (Go)

- **Server (cmd/meshsim)**
  - Serves REST endpoints for loading topologies, listing nodes, and sending messages
  - Maintains an in‐memory simulation of nodes, each with a NaCl keypair
  - Uses Gorilla Mux for routing and Gorilla WebSocket for real‐time event streaming

## Frontend (web/static)

- Single HTML page (`index.html`) with embedded CSS layouts
- D3.js‐powered SVG for force‐directed graph visualization (`app.js`)
- Controls for:
  - Selecting existing topologies
  - Creating custom topologies (paste YAML or "build via form")
  - Adjusting drop rate & TTL
  - Sending test messages

## Interface Layout

When you start the server, visit `http://localhost:8080/` in your browser:

- **Left Pane**: Controls (topology selection, drop‐rate slider, TTL slider, sender/recipient)
- **Right Pane** (vertically split):
  - Visualization (2/3 height): D3 force‐directed graph of the mesh
  - Event Log (1/3 height): Scrolling log of real‐time events

## Project Structure

```
.
├── cmd
│   └── meshsim
│       ├── main.go           # Go server entrypoint
│       └── go.mod / go.sum
├── internal
│   ├── crypto               # NaCl box keypair generation & Encrypt/Decrypt
│   ├── network             # Mesh‐network logic
│   ├── node                # Node‐level goroutines, WebSocket events
│   └── ws                  # Gorilla WebSocket Hub
├── pkg
│   └── config              # YAML loading & Topology struct
├── testdata
│   ├── clustered-12.yaml
│   ├── fullmesh-10.yaml
│   ├── linear-20.yaml
│   └── ...                 # Sample topology files
├── web
│   └── static
│       ├── index.html      # Single‐page UI
│       └── app.js          # Front‐end logic
├── README.md
└── go.mod / go.sum
```

## Prerequisites

- Go 1.24+ (any recent Go 1.x should work)
- Node.js / npm (optional, only for manual front‐end dependency installation)
- Modern browser (Chrome, Firefox, Safari, Edge)

All third-party Go modules are fetched automatically via `go mod tidy` / `go mod download`.

## Getting Started

1. Clone the repository:
```bash
git clone https://github.com/your‐username/mesh‐sim.git
cd mesh‐sim
```

2. Update Go modules:
```bash
go mod tidy
```

3. Run the server:
```bash
go run cmd/meshsim/main.go
```

You should see:
```
Server is running at http://localhost:8080
→ WebSocket endpoint: ws://localhost:8080/ws
```

4. Open your browser and navigate to: `http://localhost:8080/`