package dispatch

import (
	"log"
	"sync"

	"github.com/example/ride-matching/internal/models"
	"github.com/gorilla/websocket"
)

// WSSession represents a connected driver session
type WSSession struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *WSSession) Send(offer models.MatchOffer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteJSON(offer)
}

// WSRegistry holds driver sessions
type WSRegistry struct {
	mu       sync.RWMutex
	sessions map[string]*WSSession
}

func NewWSRegistry() *WSRegistry { return &WSRegistry{sessions: make(map[string]*WSSession)} }

func (r *WSRegistry) Add(driverID string, conn *websocket.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[driverID] = &WSSession{conn: conn}
}

func (r *WSRegistry) Offer(driverID string, offer models.MatchOffer) error {
	r.mu.RLock()
	s, ok := r.sessions[driverID]
	r.mu.RUnlock()
	if !ok {
		return ErrNoSession
	}
	if err := s.Send(offer); err != nil {
		log.Printf("ws send error: %v", err)
		return err
	}
	return nil
}

var ErrNoSession = &NoSessionError{}

type NoSessionError struct{}

func (n *NoSessionError) Error() string { return "no ws session" }
