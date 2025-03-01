package main

import (
	"encoding/json"
	"fmt"
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

func NewGameSession(id string) *GameSession {
	return &GameSession{
		id: id,
		state: &GameState{
			puckX:  0,
			puckY:  0,
			puckVX: 1,
			puckVY: 1,
		},
		ticker: time.NewTicker(16 * time.Millisecond),
		doneCh: make(chan bool),
	}
}

func (gs *GameSession) Start() {
	go func() {
		for {
			select {
			case <-gs.doneCh:
				gs.ticker.Stop()
				fmt.Printf("Game session stopped: %s", gs.id)
				return
			case <-gs.ticker.C:
				gs.Update()
				gs.BroadcastState()
			}
		}
	}()
}

func (gs *GameSession) Update() {
	gs.state.puckX += gs.state.puckVX
	gs.state.puckY += gs.state.puckVY

	if gs.state.puckX > 100 || gs.state.puckX < -100 {
		gs.state.puckVX = -gs.state.puckVX
	}

	if gs.state.puckY > 100 || gs.state.puckY < -100 {
		gs.state.puckVY = -gs.state.puckVY
	}
}

func (gs *GameSession) BroadcastState() {
	if gs.state.playerA == nil || gs.state.playerB == nil {
		return
	}

	msg := struct {
		MsgType  string  `json:"type"`
		PuckX    float64 `json:"puckX"`
		PuckY    float64 `json:"puckY"`
		PlayerAX float64 `json:"playerAX"`
		PlayerAY float64 `json:"playerAY"`
		PlayerBX float64 `json:"playerBX"`
		PlayerBY float64 `json:"playerBY"`
	}{
		MsgType:  "state_update",
		PuckX:    gs.state.puckX,
		PuckY:    gs.state.puckY,
		PlayerAX: gs.state.playerA.positionX,
		PlayerAY: gs.state.playerA.positionY,
		PlayerBX: gs.state.playerB.positionX,
		PlayerBY: gs.state.playerB.positionY,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Error marshalling state update message: %s", err)
		return
	}

	_ = gs.state.playerA.conn.WriteMessage(websocket.TextMessage, data)
	_ = gs.state.playerB.conn.WriteMessage(websocket.TextMessage, data)
}

func (gs *GameSession) HandlePlayerInput(playerId string, msg []byte) {
	var input struct {
		MsgType string  `json:"type"`
		X       float64 `json:"x"`
		Y       float64 `json:"y"`
	}

	err := json.Unmarshal(msg, &input)
	if err != nil {
		fmt.Printf("Error unmarshalling player input: %s", err)
		return
	}

	if input.MsgType == "player_move" {
		switch playerId {
		case gs.state.playerA.id:
			gs.state.playerA.positionX = input.X
			gs.state.playerA.positionY = input.Y
		case gs.state.playerB.id:
			gs.state.playerB.positionX = input.X
			gs.state.playerB.positionY = input.Y
		}
	}
}

func (gs *GameSession) RemovePlayer(playerID string) {
	if gs.state.playerA != nil && gs.state.playerA.id == playerID {
		gs.state.playerA = nil
	}
	if gs.state.playerB != nil && gs.state.playerB.id == playerID {
		gs.state.playerB = nil
	}
}