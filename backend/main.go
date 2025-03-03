// Package main implements a WebSocket-based hockey game server.
// It provides HTTP endpoints for creating and joining game sessions.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Global session manager
var manager = NewManager()

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections by default
	},
}

func main() {
	r := mux.NewRouter()

	// Serve static files
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// API endpoints
	r.HandleFunc("/create", handleCreate).Methods("POST")
	r.HandleFunc("/play/{sessionID}", handlePlay)

	fmt.Println("WebHockey server running on :8080 ...")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", r))
}

// handleCreate processes requests to create a new game session.
// It returns a JSON response with the new session ID.
func handleCreate(w http.ResponseWriter, r *http.Request) {
	sessionID := manager.Create()

	fmt.Printf("New game session created: %s\n", sessionID)

	response := map[string]string{
		"sessionID": sessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handlePlay upgrades an HTTP connection to WebSocket and connects
// the client to the specified game session.
func handlePlay(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	session, ok := manager.Get(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v\n", err)
		http.Error(w, "Error during WebSocket upgrade", http.StatusBadRequest)
		return
	}

	playerID := uuid.NewString()
	player := &Player{
		ID:   playerID,
		Conn: conn,
	}

	// Assign the player to an available slot
	if session.state.PlayerA == nil {
		session.state.PlayerA = player
		fmt.Printf("Player A (ID: %s) joined session %s\n", playerID, sessionID)
	} else if session.state.PlayerB == nil {
		session.state.PlayerB = player
		fmt.Printf("Player B (ID: %s) joined session %s\n", playerID, sessionID)
	} else {
		log.Printf("Session %s is full, rejecting player\n", sessionID)
		conn.Close()
		http.Error(w, "Session full", http.StatusConflict)
		return
	}

	// Send initial dimensions to the player
	session.SendInitialDimensions(player)

	// Start listening for player messages
	go listenToPlayer(session, player)
}

// listenToPlayer handles WebSocket messages from a connected player.
// It runs in a separate goroutine for each connected player.
func listenToPlayer(s *Session, p *Player) {
	defer p.Conn.Close()

	for {
		msgType, msg, err := p.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading from player %s: %v\n", p.ID, err)
			s.RemovePlayer(p.ID)
			return
		}

		if msgType == websocket.TextMessage {
			s.HandleInput(p.ID, msg)
		}
	}
}
