# WebHockey

WebHockey is a real-time multiplayer hockey game implemented using WebSockets. Players control virtual hockey players in a browser-based interface and compete against each other to score goals.

## Overview

WebHockey provides a seamless real-time gaming experience through a WebSocket connection between the client and server. Players can create new game sessions or join existing ones via shareable session IDs.

## Backend Design

The backend is built in Go and follows a clean, modular architecture:

### Components

1. **Session Management**
   - `Manager`: Handles the lifecycle of game sessions (creation, retrieval, removal)
   - `Session`: Represents an individual game session between two players

2. **Game Physics**
   - Real-time collision detection between players and puck
   - Puck movement and bouncing mechanics
   - Goal detection and scoring

3. **WebSocket Communication**
   - Bidirectional real-time updates between server and clients
   - Player input handling
   - Game state broadcasting

### Key Design Patterns

- **Actor Model**: Each session operates as an independent actor with its own state and game loop
- **Event-driven Architecture**: The system responds to various events (player movements, collisions, goals)
- **Thread-safe Resource Management**: Concurrent access to shared resources is protected by mutexes

## Go Language Features

WebHockey leverages several powerful Go features that distinguish it from implementations in other languages:

### Goroutines and Concurrency

- **Lightweight Thread Management**: Each game session and player connection runs in its own goroutine
- **Channels for Communication**: `done` channels are used for clean termination signals
- **Tickers**: Time-based game loop implemented efficiently with `time.Ticker`

### Memory Safety

- **Garbage Collection**: Automatic memory management without manual allocation/deallocation
- **Value Semantics**: Clear ownership of data with explicit pointers where needed
- **Zero Values**: Safe initialization of structs with sensible defaults

### Type System

- **Struct Embedding**: Composition over inheritance for clear data modeling
- **Interface Satisfaction**: Implicit interface implementation for WebSocket handlers
- **Type Safety**: Strong typing prevents common runtime errors

### Standard Library

- **net/http**: Built-in HTTP server with no external dependencies
- **encoding/json**: Native JSON marshaling/unmarshaling
- **sync**: Thread-safe constructs like Mutex

### External Libraries

- **gorilla/websocket**: Production-ready WebSocket implementation
- **gorilla/mux**: Flexible HTTP routing
- **google/uuid**: Unique identifier generation for sessions and players

## Getting Started

### Prerequisites

- Go 1.15 or higher
- Modern web browser with WebSocket support

### Running the Server

```bash
cd /b:/Projekte/WebHockey/backend
go run .
```

The server will start listening on port 8080 (http://localhost:8080).

### Creating a Game

1. Make a POST request to `/create` to get a session ID
2. Connect to `/play/{sessionID}` via WebSocket from two different browsers/tabs
3. Start playing!

## Project Structure

```
/backend
  ├── main.go       # HTTP server setup and WebSocket handling
  ├── manager.go    # Session management and lifecycle
  ├── session.go    # Game logic, physics, and state management
  └── README.md     # This file
```

## Future Enhancements

- Player authentication and persistent user accounts
- Match history and statistics tracking
- Multiple game rooms with spectator mode
- Advanced game mechanics (power-ups, penalties)