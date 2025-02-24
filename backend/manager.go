package main

import (
	"sync"

	"github.com/google/uuid"
)

type SessionManager struct {
	sessions map[string]*GameSession
	mu       sync.Mutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*GameSession),
	}
}

func (sm *SessionManager) CreateSession() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := uuid.NewString()
	session := NewGameSession(id)
	sm.sessions[id] = session

	session.Start()

	return id
}

func (sm *SessionManager) GetSession(id string) (*GameSession, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	return session, ok
}

func (sm *SessionManager) RemoveSession(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, ok := sm.sessions[id]; ok {
		session.doneCh <- true
		delete(sm.sessions, id)
	}
}
