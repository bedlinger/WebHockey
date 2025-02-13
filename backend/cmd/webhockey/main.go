package main

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/bedlinger/webhockey/internal/hockey"
)

var games = make(map[string]*hockey.Game)

func findGameById(id string) *hockey.Game {
	return games[id]
}

func main() {
	http.HandleFunc("/create", createGame)
	http.HandleFunc("/play", playGame)
	log.Println("HTTP server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func createGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	game := hockey.NewGame(uuid.NewString())
	games[game.GameId] = game
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"gameId":"` + game.GameId + `"}`))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func playGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	gameId := r.URL.Path[len("/play/"):]
	game := findGameById(gameId)
	if game == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	playerName := r.URL.Query().Get("name")
	if playerName == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade:", err)
		return
	}
	player := hockey.NewPlayer(playerName, conn)
	game.AddPlayer(player)
	if game.Player2 != nil {
		game.Start()
	}
}
