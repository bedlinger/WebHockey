package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

// Player represents a connected client
type Player struct {
	ID   string
	Conn *websocket.Conn
	PosX float64
	PosY float64
}

// State contains the current game state
type State struct {
	Width      float64
	Height     float64
	GoalWidth  float64
	GoalHeight float64

	PuckX float64
	PuckY float64
	VelX  float64
	VelY  float64

	ScoreA int
	ScoreB int

	PlayerA *Player
	PlayerB *Player
}

// Session manages a single game instance
type Session struct {
	ID     string
	state  *State
	ticker *time.Ticker
	done   chan bool
}

func NewSession(id string) *Session {
	return &Session{
		ID: id,
		state: &State{
			Width:      800,
			Height:     400,
			GoalWidth:  60,
			GoalHeight: 120,
			PuckX:      400,
			PuckY:      200,
			VelX:       0,
			VelY:       0,
		},
		ticker: time.NewTicker(16 * time.Millisecond),
		done:   make(chan bool),
	}
}

// Start begins the game loop
func (s *Session) Start() {
	go func() {
		for {
			select {
			case <-s.done:
				s.ticker.Stop()
				fmt.Printf("Game session stopped: %s", s.ID)
				return
			case <-s.ticker.C:
				s.update()
				s.broadcast()
			}
		}
	}()
}

func (s *Session) update() {
	// If both players are present and puck is stationary, start movement
	if s.state.PlayerA != nil && s.state.PlayerB != nil &&
		s.state.VelX == 0 && s.state.VelY == 0 {
		s.startPuckMovement()
	}

	s.state.PuckX += s.state.VelX
	s.state.PuckY += s.state.VelY

	if s.state.PlayerA != nil {
		s.handlePlayerPuckCollision(s.state.PlayerA)
	}
	if s.state.PlayerB != nil {
		s.handlePlayerPuckCollision(s.state.PlayerB)
	}

	// Check for goals
	if s.state.PuckY >= (s.state.Height-s.state.GoalHeight)/2 &&
		s.state.PuckY <= (s.state.Height+s.state.GoalHeight)/2 {
		// Goal for player B
		if s.state.PuckX <= 0 {
			s.state.ScoreB++
			s.resetPuck()
		}
		// Goal for player A
		if s.state.PuckX >= s.state.Width {
			s.state.ScoreA++
			s.resetPuck()
		}
	}

	// Bounce off top and bottom walls
	if s.state.PuckY > s.state.Height || s.state.PuckY < 0 {
		s.state.VelY = -s.state.VelY
	}

	// Bounce off side walls (only if not in goal area)
	if s.state.PuckY < (s.state.Height-s.state.GoalHeight)/2 ||
		s.state.PuckY > (s.state.Height+s.state.GoalHeight)/2 {
		if s.state.PuckX > s.state.Width || s.state.PuckX < 0 {
			s.state.VelX = -s.state.VelX
		}
	}

	// check if the game is over
	if s.state.ScoreA == 20 || s.state.ScoreB == 20 {
		s.notifyGameOver()
		s.done <- true
	}
}

func (s *Session) notifyGameOver() {
	winner := "Player A"
	if s.state.ScoreB > s.state.ScoreA {
		winner = "Player B"
	}

	msg := struct {
		MsgType string `json:"type"`
		Message string `json:"message"`
		Winner  string `json:"winner"`
		ScoreA  int    `json:"scoreA"`
		ScoreB  int    `json:"scoreB"`
	}{
		MsgType: "game_over",
		Message: "Game Over!",
		Winner:  winner,
		ScoreA:  s.state.ScoreA,
		ScoreB:  s.state.ScoreB,
	}

	data, _ := json.Marshal(msg)
	if s.state.PlayerA != nil {
		s.state.PlayerA.Conn.WriteMessage(websocket.TextMessage, data)
	}
	if s.state.PlayerB != nil {
		s.state.PlayerB.Conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (s *Session) handlePlayerPuckCollision(player *Player) {
	dx := s.state.PuckX - player.PosX
	dy := s.state.PuckY - player.PosY
	distance := math.Sqrt(dx*dx + dy*dy)

	if distance < 30 { // 20 + 10 = combined radii
		// Normalize collision vector
		nx := dx / distance
		ny := dy / distance

		s.state.PuckX = player.PosX + (30 * nx)
		s.state.PuckY = player.PosY + (30 * ny)

		speedFactor := 10.0 // Adjust this value to control bounce strength
		s.state.VelX = nx * speedFactor
		s.state.VelY = ny * speedFactor

		maxSpeed := 15.0 // Adjust this value to control maximum puck speed
		currentSpeed := math.Sqrt(s.state.VelX*s.state.VelX + s.state.VelY*s.state.VelY)
		if currentSpeed > maxSpeed {
			ratio := maxSpeed / currentSpeed
			s.state.VelX *= ratio
			s.state.VelY *= ratio
		}
	}
}

func (s *Session) resetPuck() {
	s.state.PuckX = s.state.Width / 2
	s.state.PuckY = s.state.Height / 2

	// Give the puck a random initial velocity
	speed := 5.0
	s.state.VelX = speed * float64(1-2*rand.Intn(2)) // Random direction
	s.state.VelY = speed * float64(1-2*rand.Intn(2)) // Random direction
}

// Add this new method to start puck movement
func (s *Session) startPuckMovement() {
	if s.state.VelX == 0 && s.state.VelY == 0 {
		s.resetPuck()
	}
}

func (s *Session) broadcast() {
	if s.state.PlayerA == nil || s.state.PlayerB == nil {
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
		FieldWidth:  s.state.Width,
		FieldHeight: s.state.Height,
		GoalWidth:   s.state.GoalWidth,
		GoalHeight:  s.state.GoalHeight,
		PuckX:       s.state.PuckX,
		PuckY:       s.state.PuckY,
		PlayerAX:    s.state.PlayerA.PosX,
		PlayerAY:    s.state.PlayerA.PosY,
		PlayerBX:    s.state.PlayerB.PosX,
		PlayerBY:    s.state.PlayerB.PosY,
		ScoreA:      s.state.ScoreA,
		ScoreB:      s.state.ScoreB,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Error marshalling state update message: %s", err)
		return
	}

	_ = s.state.PlayerA.Conn.WriteMessage(websocket.TextMessage, data)
	_ = s.state.PlayerB.Conn.WriteMessage(websocket.TextMessage, data)
}

func (s *Session) HandleInput(playerID string, msg []byte) {
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
		switch playerID {
		case s.state.PlayerA.ID:
			s.state.PlayerA.PosX = input.X
			s.state.PlayerA.PosY = input.Y
		case s.state.PlayerB.ID:
			s.state.PlayerB.PosX = input.X
			s.state.PlayerB.PosY = input.Y
		}
	}
}

func (s *Session) RemovePlayer(playerID string) {
	if s.state.PlayerA != nil && s.state.PlayerA.ID == playerID {
		s.state.PlayerA = nil
	}
	if s.state.PlayerB != nil && s.state.PlayerB.ID == playerID {
		s.state.PlayerB = nil
	}

	// Check if both players are gone
	if s.state.PlayerA == nil && s.state.PlayerB == nil {
		s.done <- true
		return
	}

	// Notify remaining player
	msg := struct {
		MsgType string `json:"type"`
		Message string `json:"message"`
	}{
		MsgType: "player_left",
		Message: "Other player has left the game",
	}

	data, _ := json.Marshal(msg)
	if s.state.PlayerA != nil {
		s.state.PlayerA.Conn.WriteMessage(websocket.TextMessage, data)
	}
	if s.state.PlayerB != nil {
		s.state.PlayerB.Conn.WriteMessage(websocket.TextMessage, data)
	}
}
