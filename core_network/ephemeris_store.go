package main

import (
	"sync"
	"time"
)

// EphemerisStore keeps the latest satellite states received from orbit_calc.
type EphemerisStore struct {
	mu   sync.RWMutex
	data []Ephemeris
	lastUpdated time.Time
}

func NewEphemerisStore() *EphemerisStore {
	return &EphemerisStore{}
}

func (s *EphemerisStore) Update(data []Ephemeris) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make([]Ephemeris, len(data))
	copy(s.data, data)
	s.lastUpdated = time.Now()
}

func (s *EphemerisStore) List() []Ephemeris {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data := make([]Ephemeris, len(s.data))
	copy(data, s.data)
	return data
}

// LastUpdate returns the time the store was last updated. Zero value if never updated.
func (s *EphemerisStore) LastUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdated
}
