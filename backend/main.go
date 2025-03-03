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

var manager = NewManager()

func main() {
	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	r.HandleFunc("/create", handleCreate).Methods("POST")
	r.HandleFunc("/play/{sessionID}", handlePlay)

	fmt.Println("Server running on :8080 ...")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", r))
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	sessionID := manager.Create()

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

	session, ok := manager.Get(sessionID)
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
		ID:   playerID,
		Conn: conn,
	}

	if session.state.PlayerA == nil {
		session.state.PlayerA = player
	} else if session.state.PlayerB == nil {
		session.state.PlayerB = player
	} else {
		http.Error(w, "Session full", http.StatusConflict)
		conn.Close()
		return
	}

	fmt.Printf("Player %s joined session %s\n", playerID, sessionID)

	go listenToPlayer(session, player)
}

func listenToPlayer(s *Session, p *Player) {
	defer p.Conn.Close()

	for {
		msgType, msg, err := p.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error while reading player %s: %v\n", p.ID, err)
			s.RemovePlayer(p.ID)
			return
		}

		if msgType == websocket.TextMessage {
			s.HandleInput(p.ID, msg)
		}
	}
}
