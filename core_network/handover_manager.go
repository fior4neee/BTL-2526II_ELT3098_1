package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrInvalidHandover    = errors.New("session not eligible for handover")
	ErrHandoverInProgress = errors.New("handover already in progress")
)

// OrbitUpdate represents data from orbit_calc module
type OrbitUpdate struct {
	SatelliteID string
	Location    Location
}

// HandoverManager handles connection FSM and transitions
type HandoverManager struct {
	sessions        map[string]*Session
	handoverHistory []HandoverEvent
	gatewayPool     *GatewayPool
	settings        SystemSettings
	mu              sync.RWMutex
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

	// Choose least-loaded alive gateway (simple policy)
	gateways := hm.gatewayPool.GetAllGateways()
	var chosenGW string
	minLoad := int(^uint(0) >> 1) // max int
	for _, g := range gateways {
		if g.Status != "alive" {
			continue
		}
		if g.CurrentSessions >= g.MaxSessions {
			continue
		}
		if g.CurrentSessions < minLoad {
			minLoad = g.CurrentSessions
			chosenGW = g.ID
		}
	}

	// fallback: pick any alive gateway
	if chosenGW == "" {
		for _, g := range gateways {
			if g.Status == "alive" {
				chosenGW = g.ID
				break
			}
		}
	}

	if chosenGW == "" {
		return nil, fmt.Errorf("no available gateways")
	}

	if err := hm.gatewayPool.UpdateLoad(chosenGW, 1); err != nil {
		return nil, err
	}

	sID := fmt.Sprintf("SES-%d", time.Now().UnixNano()%1000000)
	session := &Session{
		ID: sID, RouterMAC: mac, SatelliteID: hm.settings.DefaultSatelliteID,
		CurrentGatewayID: chosenGW, State: StateHold, StartTime: time.Now(),
	}
	hm.sessions[sID] = session
	hm.gatewayPool.LogTelemetry("session_create", chosenGW, session, "new session created")
	return session, nil
}

// HandleRouterDisconnect removes an active session and releases gateway capacity
func (hm *HandoverManager) HandleRouterDisconnect(sessionID string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	session, ok := hm.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	from := session.CurrentGatewayID
	if from != "" {
		if err := hm.gatewayPool.UpdateLoad(from, -1); err != nil {
			return err
		}
	}

	delete(hm.sessions, sessionID)
	hm.gatewayPool.LogTelemetry("session_disconnect", from, session, "session removed")
	return nil
}

// ClearAllSessions force-removes all active sessions and resets gateway loads.
// Intended for admin/testing use only.
func (hm *HandoverManager) ClearAllSessions() int {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	count := 0
	for id, session := range hm.sessions {
		if session.CurrentGatewayID != "" {
			_ = hm.gatewayPool.UpdateLoad(session.CurrentGatewayID, -1)
			hm.gatewayPool.LogTelemetry("session_cleared", session.CurrentGatewayID, session, "cleared by admin")
		}
		delete(hm.sessions, id)
		count++
	}
	return count
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

// GetHandoverHistory returns recent handover events, newest first.
func (hm *HandoverManager) GetHandoverHistory() []HandoverEvent {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	res := make([]HandoverEvent, len(hm.handoverHistory))
	copy(res, hm.handoverHistory)
	return res
}

// TriggerManualHandover validates and completes a demo handover for a session.
func (hm *HandoverManager) TriggerManualHandover(sessionID, targetGatewayID string) (*HandoverEvent, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	session, ok := hm.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	if session.State == StatePrepare || session.State == StateExecute {
		return nil, ErrHandoverInProgress
	}
	if targetGatewayID == "" || targetGatewayID == session.CurrentGatewayID {
		return nil, ErrInvalidHandover
	}
	targetGateway, ok := hm.gatewayPool.GetGateway(targetGatewayID)
	if !ok || targetGateway.Status != "alive" {
		return nil, ErrInvalidHandover
	}

	fromGatewayID := session.CurrentGatewayID
	if err := hm.gatewayPool.UpdateLoad(targetGatewayID, 1); err != nil {
		return nil, err
	}
	if err := hm.gatewayPool.UpdateLoad(fromGatewayID, -1); err != nil {
		_ = hm.gatewayPool.UpdateLoad(targetGatewayID, -1)
		return nil, err
	}

	session.State = StateExecute
	session.NextGatewayID = targetGatewayID

	event := HandoverEvent{
		ID:          fmt.Sprintf("HO-%d", time.Now().UnixNano()),
		Timestamp:   time.Now(),
		SessionID:   session.ID,
		FromGateway: fromGatewayID,
		ToGateway:   targetGatewayID,
		DurationMs:  75,
		PacketLoss:  0.0005,
		Success:     true,
	}

	session.CurrentGatewayID = targetGatewayID
	session.NextGatewayID = ""
	session.State = StateHold

	hm.handoverHistory = append([]HandoverEvent{event}, hm.handoverHistory...)
	if len(hm.handoverHistory) > 100 {
		hm.handoverHistory = hm.handoverHistory[:100]
	}

	hm.gatewayPool.LogTelemetry("handover", targetGatewayID, event, "manual handover completed")
	return &event, nil
}
