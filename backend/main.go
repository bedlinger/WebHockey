package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Willkommen beim Web-Hockey!")
	})

	fmt.Println("Server läuft auf Port 8080 ...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}