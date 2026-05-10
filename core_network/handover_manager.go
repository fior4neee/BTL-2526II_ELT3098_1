package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// OrbitUpdate represents real-time ephemeris data pushed from orbit_calc (Python)
type OrbitUpdate struct {
	SatelliteID string
	Location    Location
	Timestamp   time.Time
}

// HandoverManager coordinates the State Machine and Session lifecycles
type HandoverManager struct {
	sessions      map[string]*Session
	history       []HandoverEvent
	gatewayPool   *GatewayPool
	OrbitUpdateCh chan OrbitUpdate // Input channel for orbital telemetry
	mu            sync.RWMutex
}

// NewHandoverManager creates a manager and starts the background worker
func NewHandoverManager(pool *GatewayPool) *HandoverManager {
	hm := &HandoverManager{
		sessions:      make(map[string]*Session),
		history:       make([]HandoverEvent, 0),
		gatewayPool:   pool,
		OrbitUpdateCh: make(chan OrbitUpdate, 1000),
	}

	go hm.ProcessOrbitUpdates()
	return hm
}

// POST /api/v1/router/connect logic
func (hm *HandoverManager) HandleRouterConnect(mac string, loc Location) (*Session, error) {
	if mac == "" {
		return nil, errors.New("missing router MAC")
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	bestGW := "GW-HAN-01" // Simple logic: default to Hanoi for initial connection

	gw, err := hm.gatewayPool.GetGateway(bestGW)
	if err != nil || gw.Status != "alive" {
		return nil, fmt.Errorf("gateway %s unavailable", bestGW)
	}

	if err := hm.gatewayPool.UpdateLoad(bestGW, 1); err != nil {
		return nil, err
	}

	sessionID := fmt.Sprintf("SES-%d", time.Now().UnixNano()%10000)
	session := &Session{
		ID:               sessionID,
		RouterMAC:        mac,
		SatelliteID:      "SAT-LEO-001",
		CurrentGatewayID: bestGW,
		State:            StateAcquire,
		StartTime:        time.Now(),
	}

	hm.sessions[sessionID] = session
	hm.gatewayPool.LogTelemetry("SESSION_CREATED", bestGW, sessionID, "New router authenticated")

	return session, nil
}

// GetActiveSessions returns all ongoing sessions for ISP monitoring
func (hm *HandoverManager) GetActiveSessions() []Session {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	res := make([]Session, 0, len(hm.sessions))
	for _, s := range hm.sessions {
		res = append(res, *s)
	}
	return res
}

// ProcessOrbitUpdates background routine to update FSM states based on satellite movement
func (hm *HandoverManager) ProcessOrbitUpdates() {
	for update := range hm.OrbitUpdateCh {
		hm.evaluateHandovers(update)
	}
}

// evaluateHandovers iterates through sessions to trigger state transitions
func (hm *HandoverManager) evaluateHandovers(update OrbitUpdate) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	for _, session := range hm.sessions {
		if session.SatelliteID != update.SatelliteID {
			continue
		}

		gw, err := hm.gatewayPool.GetGateway(session.CurrentGatewayID)
		if err != nil {
			continue
		}

		currentElev, _, err := hm.gatewayPool.CalculateElevation(session.CurrentGatewayID, update.Location)
		if err != nil {
			continue
		}

		// --- FSM Transitions ---
		switch session.State {
		case StateAcquire:
			if currentElev > gw.MinElevationDeg {
				session.State = StateHold
			}
		case StateHold:
			if currentElev <= gw.MinElevationDeg+ElevationHysteresis {
				session.State = StatePrepare
				nextGW, err := hm.predictNextGateway(update.Location)
				if err == nil {
					session.NextGatewayID = nextGW
				}
			}
		case StatePrepare:
			if currentElev < gw.MinElevationDeg {
				if session.NextGatewayID != "" {
					session.State = StateExecute
					go hm.executeHandoverGoroutine(session, session.NextGatewayID)
				} else {
					session.State = StateAcquire
				}
			}
		}
	}
}

// predictNextGateway finds the gateway with the highest visibility for a given satellite position
func (hm *HandoverManager) predictNextGateway(satLoc Location) (string, error) {
	var bestGW string
	var maxElev float64 = -1.0
	allGateways := hm.gatewayPool.GetAllGateways()

	for _, gw := range allGateways {
		if gw.Status != "alive" {
			continue
		}
		elev, _, err := hm.gatewayPool.CalculateElevation(gw.ID, satLoc)
		if err == nil && elev > maxElev && elev >= gw.MinElevationDeg {
			maxElev = elev
			bestGW = gw.ID
		}
	}

	if bestGW == "" {
		return "", errors.New("no coverage found")
	}
	return bestGW, nil
}

// executeHandoverGoroutine migrates a session between gateways asynchronously
func (hm *HandoverManager) executeHandoverGoroutine(session *Session, nextGW string) error {
	oldGW := session.CurrentGatewayID

	if err := hm.gatewayPool.UpdateLoad(nextGW, 1); err != nil {
		session.State = StatePrepare
		return err
	}

	_ = hm.gatewayPool.UpdateLoad(oldGW, -1)
	session.CurrentGatewayID = nextGW
	session.State = StateHold

	hm.mu.Lock()
	hm.history = append(hm.history, HandoverEvent{
		EventID:   fmt.Sprintf("HO-%d", time.Now().UnixNano()%100000),
		SessionID: session.ID, OldGW: oldGW, NewGW: nextGW, Timestamp: time.Now(),
	})
	hm.mu.Unlock()

	hm.gatewayPool.LogTelemetry("HANDOVER_SUCCESS", nextGW, session.ID, "Handover completed")
	return nil
}
