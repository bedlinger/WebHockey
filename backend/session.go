// Package main implements a WebSocket-based hockey game server.
// It manages game sessions, player connections, and game state.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

// Player represents a connected client in the game.
type Player struct {
	ID   string          // Unique identifier for the player
	Conn *websocket.Conn // WebSocket connection to the client
	PosX float64         // X position of the player on the field
	PosY float64         // Y position of the player on the field
}

// State contains the current game state including field dimensions,
// puck position, velocities, and player scores.
type State struct {
	Width      float64 // Width of the playing field
	Height     float64 // Height of the playing field
	GoalWidth  float64 // Width of the goal
	GoalHeight float64 // Height of the goal

	PuckX float64 // X position of the puck
	PuckY float64 // Y position of the puck
	VelX  float64 // X velocity of the puck
	VelY  float64 // Y velocity of the puck

	ScoreA int // Score of player A
	ScoreB int // Score of player B

	PlayerA *Player // First player
	PlayerB *Player // Second player
}

// Session manages a single game instance between two players.
type Session struct {
	ID     string       // Unique identifier for the session
	state  *State       // Current game state
	ticker *time.Ticker // Game loop ticker
	done   chan bool    // Channel to signal session termination
}

// NewSession creates and initializes a new game session with the specified ID.
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
		ticker: time.NewTicker(16 * time.Millisecond), // ~60 FPS
		done:   make(chan bool),
	}
}

// Start begins the game loop that updates state and broadcasts to players.
func (s *Session) Start() {
	go func() {
		for {
			select {
			case <-s.done:
				s.ticker.Stop()
				fmt.Printf("Game session stopped: %s\n", s.ID)
				return
			case <-s.ticker.C:
				s.update()
				s.broadcast()
			}
		}
	}()
}

// update processes one step of game physics including puck movement,
// collisions, goals, and game end conditions.
func (s *Session) update() {
	// If both players are present and puck is stationary, start movement
	if s.state.PlayerA != nil && s.state.PlayerB != nil &&
		s.state.VelX == 0 && s.state.VelY == 0 {
		s.startPuckMovement()
	}

	s.state.PuckX += s.state.VelX
	s.state.PuckY += s.state.VelY

	// Handle player-puck collisions
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

	// Check if the game is over
	if s.state.ScoreA == 20 || s.state.ScoreB == 20 {
		s.notifyGameOver()
		s.done <- true
	}
}

// notifyGameOver sends a game over message to all connected players
// including the final scores and winner.
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

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Error marshalling game over message: %s\n", err)
		return
	}

	if s.state.PlayerA != nil {
		if err := s.state.PlayerA.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Printf("Error sending game over to Player A: %s\n", err)
		}
	}
	if s.state.PlayerB != nil {
		if err := s.state.PlayerB.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Printf("Error sending game over to Player B: %s\n", err)
		}
	}
}

// handlePlayerPuckCollision calculates and applies physics when a player
// collides with the puck.
func (s *Session) handlePlayerPuckCollision(player *Player) {
	const (
		collisionRadius = 30.0 // Combined radius of player and puck
		speedFactor     = 10.0 // Controls bounce strength
		maxSpeed        = 30.0 // Maximum puck speed
	)

	dx := s.state.PuckX - player.PosX
	dy := s.state.PuckY - player.PosY
	distance := math.Sqrt(dx*dx + dy*dy)

	if distance < collisionRadius {
		// Normalize collision vector
		nx := dx / distance
		ny := dy / distance

		// Reposition puck to avoid overlap
		s.state.PuckX = player.PosX + (collisionRadius * nx)
		s.state.PuckY = player.PosY + (collisionRadius * ny)

		// Apply velocity based on collision direction
		s.state.VelX = nx * speedFactor
		s.state.VelY = ny * speedFactor

		// Limit puck speed
		currentSpeed := math.Sqrt(s.state.VelX*s.state.VelX + s.state.VelY*s.state.VelY)
		if currentSpeed > maxSpeed {
			ratio := maxSpeed / currentSpeed
			s.state.VelX *= ratio
			s.state.VelY *= ratio
		}
	}
}

// resetPuck positions the puck in the center of the field
// and gives it a random initial velocity.
func (s *Session) resetPuck() {
	s.state.PuckX = s.state.Width / 2
	s.state.PuckY = s.state.Height / 2

	// Give the puck a random initial velocity
	speed := 5.0
	s.state.VelX = speed * float64(1-2*rand.Intn(2)) // Random direction
	s.state.VelY = speed * float64(1-2*rand.Intn(2)) // Random direction
}

// startPuckMovement initializes puck movement if it's currently stationary.
func (s *Session) startPuckMovement() {
	if s.state.VelX == 0 && s.state.VelY == 0 {
		s.resetPuck()
	}
}

// broadcast sends the current game state to all connected players.
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
		fmt.Printf("Error marshalling state update message: %s\n", err)
		return
	}

	if err := s.state.PlayerA.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		fmt.Printf("Error sending state to Player A: %s\n", err)
	}

	if err := s.state.PlayerB.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		fmt.Printf("Error sending state to Player B: %s\n", err)
	}
}

// HandleInput processes player movement commands from clients.
func (s *Session) HandleInput(playerID string, msg []byte) {
	var input struct {
		MsgType string  `json:"type"`
		X       float64 `json:"x"`
		Y       float64 `json:"y"`
	}

	err := json.Unmarshal(msg, &input)
	if err != nil {
		fmt.Printf("Error unmarshalling player input: %s\n", err)
		return
	}

	if input.MsgType == "player_move" {
		switch playerID {
		case s.state.PlayerA.ID:
			// Player A is restricted to the left half of the field
			if input.X <= s.state.Width/2 {
				s.state.PlayerA.PosX = input.X
				s.state.PlayerA.PosY = input.Y
			} else {
				// If player tries to move beyond their half, restrict to the halfway line
				s.state.PlayerA.PosX = s.state.Width / 2
				s.state.PlayerA.PosY = input.Y
			}
		case s.state.PlayerB.ID:
			// Player B is restricted to the right half of the field
			if input.X >= s.state.Width/2 {
				s.state.PlayerB.PosX = input.X
				s.state.PlayerB.PosY = input.Y
			} else {
				// If player tries to move beyond their half, restrict to the halfway line
				s.state.PlayerB.PosX = s.state.Width / 2
				s.state.PlayerB.PosY = input.Y
			}
		}
	}
}

// RemovePlayer handles disconnection of a player from the session
// and notifies the remaining player.
func (s *Session) RemovePlayer(playerID string) {
	if s.state.PlayerA != nil && s.state.PlayerA.ID == playerID {
		s.state.PlayerA = nil
		fmt.Printf("Player A (ID: %s) removed from session %s\n", playerID, s.ID)
	}
	if s.state.PlayerB != nil && s.state.PlayerB.ID == playerID {
		s.state.PlayerB = nil
		fmt.Printf("Player B (ID: %s) removed from session %s\n", playerID, s.ID)
	}

	// Check if both players are gone
	if s.state.PlayerA == nil && s.state.PlayerB == nil {
		fmt.Printf("All players left session %s. Terminating session.\n", s.ID)
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

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Error marshalling player left message: %s\n", err)
		return
	}

	if s.state.PlayerA != nil {
		if err := s.state.PlayerA.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Printf("Error notifying Player A of disconnect: %s\n", err)
		}
	}
	if s.state.PlayerB != nil {
		if err := s.state.PlayerB.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Printf("Error notifying Player B of disconnect: %s\n", err)
		}
	}
}

// SendInitialDimensions sends the game dimensions to a newly connected player.
func (s *Session) SendInitialDimensions(player *Player) {
	msg := struct {
		MsgType     string  `json:"type"`
		FieldWidth  float64 `json:"fieldWidth"`
		FieldHeight float64 `json:"fieldHeight"`
		GoalWidth   float64 `json:"goalWidth"`
		GoalHeight  float64 `json:"goalHeight"`
	}{
		MsgType:     "init_dimensions",
		FieldWidth:  s.state.Width,
		FieldHeight: s.state.Height,
		GoalWidth:   s.state.GoalWidth,
		GoalHeight:  s.state.GoalHeight,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Error marshalling initial dimensions message: %s\n", err)
		return
	}

	if err := player.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		fmt.Printf("Error sending initial dimensions to player: %s\n", err)
	}
}
