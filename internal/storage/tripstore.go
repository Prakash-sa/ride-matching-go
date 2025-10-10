package storage

import (
	"sync"

	"github.com/example/ride-matching/internal/models"
)

// TripStore defines persistence operations for rides.
type TripStore interface {
	SaveRide(r *models.Ride) error
	UpdateRide(r *models.Ride) error
}

type MemoryStore struct {
	mu    sync.RWMutex
	rides map[string]*models.Ride
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rides: make(map[string]*models.Ride)}
}

func (m *MemoryStore) SaveRide(r *models.Ride) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rides[r.ID] = r
	return nil
}

func (m *MemoryStore) UpdateRide(r *models.Ride) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rides[r.ID] = r
	return nil
}

func (m *MemoryStore) Get(id string) (*models.Ride, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rides[id]
	return r, ok
}
