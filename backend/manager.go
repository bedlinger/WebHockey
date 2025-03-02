package main

import (
	"sync"

	"github.com/google/uuid"
)

// Manager handles game session lifecycle and storage
type Manager struct {
	sessions map[string]*Session
	mu       sync.Mutex
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// Create initializes a new game session and returns its ID
func (m *Manager) Create() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.NewString()
	session := NewSession(id)
	m.sessions[id] = session

	session.Start()
	return id
}

// Get returns a session by ID if it exists
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	return session, ok
}

// Remove terminates and removes a session
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		session.done <- true
		delete(m.sessions, id)
	}
}
