package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
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
	fieldWidth  float64
	fieldHeight float64
	goalWidth   float64
	goalHeight  float64

	puckX  float64
	puckY  float64
	puckVX float64
	puckVY float64

	scoreA int
	scoreB int

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
			fieldWidth:  800, // pixels
			fieldHeight: 400, // pixels
			goalWidth:   60,  // pixels
			goalHeight:  120, // pixels
			puckX:       400,
			puckY:       200,
			puckVX:      5,
			puckVY:      5,
			scoreA:      0,
			scoreB:      0,
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

	// Check for goals
	if gs.state.puckY >= (gs.state.fieldHeight-gs.state.goalHeight)/2 &&
		gs.state.puckY <= (gs.state.fieldHeight+gs.state.goalHeight)/2 {
		// Goal for player B
		if gs.state.puckX <= 0 {
			gs.state.scoreB++
			gs.resetPuck()
		}
		// Goal for player A
		if gs.state.puckX >= gs.state.fieldWidth {
			gs.state.scoreA++
			gs.resetPuck()
		}
	}

	// Bounce off top and bottom walls
	if gs.state.puckY > gs.state.fieldHeight || gs.state.puckY < 0 {
		gs.state.puckVY = -gs.state.puckVY
	}

	// Bounce off side walls (only if not in goal area)
	if gs.state.puckY < (gs.state.fieldHeight-gs.state.goalHeight)/2 ||
		gs.state.puckY > (gs.state.fieldHeight+gs.state.goalHeight)/2 {
		if gs.state.puckX > gs.state.fieldWidth || gs.state.puckX < 0 {
			gs.state.puckVX = -gs.state.puckVX
		}
	}
}

func (gs *GameSession) resetPuck() {
	gs.state.puckX = gs.state.fieldWidth / 2
	gs.state.puckY = gs.state.fieldHeight / 2
	gs.state.puckVX = 5 * float64(1-2*rand.Intn(2)) // Random direction
	gs.state.puckVY = 5 * float64(1-2*rand.Intn(2)) // Random direction
}

func (gs *GameSession) BroadcastState() {
	if gs.state.playerA == nil || gs.state.playerB == nil {
		return
	}

	msg := struct {
		MsgType     string  `json:"type"`
		FieldWidth  float64 `json:"fieldWidth"`
		FieldHeight float64 `json:"fieldHeight"`
		GoalWidth   float64 `json:"goalWidth"`
		GoalHeight  float64 `json:"goalHeight"`
		PuckX       float64 `json:"puckX"`
		PuckY       float64 `json:"puckY"`
		PlayerAX    float64 `json:"playerAX"`
		PlayerAY    float64 `json:"playerAY"`
		PlayerBX    float64 `json:"playerBX"`
		PlayerBY    float64 `json:"playerBY"`
		ScoreA      int     `json:"scoreA"`
		ScoreB      int     `json:"scoreB"`
	}{
		MsgType:     "state_update",
		FieldWidth:  gs.state.fieldWidth,
		FieldHeight: gs.state.fieldHeight,
		GoalWidth:   gs.state.goalWidth,
		GoalHeight:  gs.state.goalHeight,
		PuckX:       gs.state.puckX,
		PuckY:       gs.state.puckY,
		PlayerAX:    gs.state.playerA.positionX,
		PlayerAY:    gs.state.playerA.positionY,
		PlayerBX:    gs.state.playerB.positionX,
		PlayerBY:    gs.state.playerB.positionY,
		ScoreA:      gs.state.scoreA,
		ScoreB:      gs.state.scoreB,
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
