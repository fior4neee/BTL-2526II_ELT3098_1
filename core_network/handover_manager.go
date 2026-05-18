package main

import (
	"fmt"
	"sync"
	"time"
)

// OrbitUpdate represents data from orbit_calc module
type OrbitUpdate struct {
	SatelliteID string
	Location    Location
}

// HandoverManager handles connection FSM and transitions
type HandoverManager struct {
	sessions    map[string]*Session
	gatewayPool *GatewayPool
	settings    SystemSettings
	mu          sync.RWMutex
}

// NewHandoverManager initializes manager with system settings
func NewHandoverManager(pool *GatewayPool, settings SystemSettings) *HandoverManager {
	return &HandoverManager{
		sessions:    make(map[string]*Session),
		gatewayPool: pool,
		settings:    settings,
	}
}

// HandleRouterConnect handles new incoming connection requests
func (hm *HandoverManager) HandleRouterConnect(mac string) (*Session, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	targetGW := "GW-HAN-01" // Mock logic
	if err := hm.gatewayPool.UpdateLoad(targetGW, 1); err != nil {
		return nil, err
	}

	sID := fmt.Sprintf("SES-%d", time.Now().UnixNano()%10000)
	session := &Session{
		ID: sID, RouterMAC: mac, SatelliteID: hm.settings.DefaultSatelliteID,
		CurrentGatewayID: targetGW, State: StateHold, StartTime: time.Now(),
	}
	hm.sessions[sID] = session
	return session, nil
}

// GetActiveSessions returns current active sessions for monitoring
func (hm *HandoverManager) GetActiveSessions() []Session {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	res := make([]Session, 0, len(hm.sessions))
	for _, s := range hm.sessions {
		res = append(res, *s)
	}
	return res
}
