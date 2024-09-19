# GoSocketServer

GoSocketServer is a simple WebSocket-based chat server written in Go using the [Gorilla WebSocket](https://github.com/gorilla/websocket) package. The server allows multiple clients to connect and send messages to each other in real time.

## Features

- Handles WebSocket connections and upgrades HTTP requests to WebSocket.
- Broadcast messages to all connected clients.
- Tracks active clients and manages connections.

## Getting Started

### Clone the repository

```bash
git git@github.com:Liam731/go_socket_server.git
cd go_socket_server
```
### Running the server

```bash
make run_server
```

### Testing the server

```bash
make test_all
```

