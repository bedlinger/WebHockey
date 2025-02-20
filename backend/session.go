package main

import (
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	id        string
	conn      *websocket.Conn
	positionX float64
	positionY float64
}

type GameState struct {
	puckX  float64
	puckY  float64
	puckVX float64
	puckVY float64

	playerA *Player
	playerB *Player
}

type GameSession struct {
	id     string
	state  *GameState
	ticker *time.Ticker
	doneCh chan bool
}
