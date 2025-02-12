package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for {
		// read message from browser
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		// print the message to the console
		log.Printf("%s sent: %s", conn.RemoteAddr(), string(msg))

		// write message back to the browser
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println(err)
			return
		}
	}
}
