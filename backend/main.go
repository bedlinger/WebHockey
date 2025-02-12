package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/ws", handleConnections)
	log.Println("HTTP server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
