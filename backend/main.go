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

var manager = NewSessionManager()

func main() {
	r := mux.NewRouter()

	// Add this line to serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	r.HandleFunc("/create", handleCreateSession).Methods("POST")
	r.HandleFunc("/play/{sessionID}", handlePlay)

	fmt.Println("Server running on :8080 ...")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func handleCreateSession(w http.ResponseWriter, r *http.Request) {
	sessionID := manager.CreateSession()

	response := map[string]string{
		"sessionID": sessionID,
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections by default
	},
}

func handlePlay(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	session, ok := manager.GetSession(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Error during WebSocket-Upgrade", http.StatusBadRequest)
		return
	}

	playerID := uuid.NewString()
	player := &Player{
		id:   playerID,
		conn: conn,
	}

	if session.state.playerA == nil {
		session.state.playerA = player
	} else if session.state.playerB == nil {
		session.state.playerB = player
	} else {
		http.Error(w, "Session full", http.StatusConflict)
		conn.Close()
		return
	}

	fmt.Printf("Player %s joined session %s\n", playerID, sessionID)

	go listenToPlayer(session, player)
}

func listenToPlayer(session *GameSession, player *Player) {
	defer player.conn.Close()

	for {
		msgType, msg, err := player.conn.ReadMessage()
		if err != nil {
			log.Printf("Fehler beim Lesen von Spieler %s: %v\n", player.id, err)
			// Remove the player from the session when an error occurs.
			session.RemovePlayer(player.id)
			return
		}

		if msgType == websocket.TextMessage {
			session.HandlePlayerInput(player.id, msg)
		}
	}
}
