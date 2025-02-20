package main

import "github.com/gorilla/websocket"

type Player struct {
	id        string
	conn      *websocket.Conn
	positionX float64
	positionY float64
}
