// Package main implements a WebSocket-based hockey game server.
package main

import (
	"sync"

	"github.com/google/uuid"
)

// Manager handles game session lifecycle and storage.
// It provides thread-safe access to create, retrieve, and remove game sessions.
type Manager struct {
	sessions map[string]*Session // Map of active game sessions by ID
	mu       sync.Mutex          // Mutex to synchronize access to sessions
}

// NewManager creates and initializes a new session manager.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// Create initializes a new game session and returns its ID.
// The session is automatically started.
func (m *Manager) Create() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.NewString()
	session := NewSession(id)
	m.sessions[id] = session

	session.Start()
	return id
}

// Get returns a session by ID if it exists.
// The boolean return value indicates whether the session was found.
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	return session, ok
}

// Remove terminates and removes a session by ID.
// If the session doesn't exist, this is a no-op.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		session.done <- true
		delete(m.sessions, id)
	}
}
