package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// OrbitUpdate represents data from orbit_calc module
type OrbitUpdate struct {
	SatelliteID string
	Location    Location
}

// HandoverManager handles connection FSM, transitions, and historic tracking
type HandoverManager struct {
	sessions        map[string]*Session
	gatewayPool     *GatewayPool
	settings        SystemSettings
	billingManager  *BillingManager
	handoverHistory []HandoverEvent
	wsClients       map[chan interface{}]bool
	registerWs      chan chan interface{}
	unregisterWs    chan chan interface{}
	mu              sync.RWMutex
}

// NewHandoverManager initializes manager with system settings, billing engine, and live telemetry channels
func NewHandoverManager(pool *GatewayPool, settings SystemSettings, bm *BillingManager) *HandoverManager {
	hm := &HandoverManager{
		sessions:        make(map[string]*Session),
		gatewayPool:     pool,
		settings:        settings,
		billingManager:  bm,
		handoverHistory: make([]HandoverEvent, 0),
		wsClients:       make(map[chan interface{}]bool),
		registerWs:      make(chan chan interface{}),
		unregisterWs:    make(chan chan interface{}),
	}

	// Start an asynchronous central hub loop to manage active WebSocket listeners safely
	go hm.startTelemetryHub()
	return &HandoverManager{
		sessions:        hm.sessions,
		gatewayPool:     hm.gatewayPool,
		settings:        hm.settings,
		billingManager:  hm.billingManager,
		handoverHistory: hm.handoverHistory,
		wsClients:       hm.wsClients,
		registerWs:      hm.registerWs,
		unregisterWs:    hm.unregisterWs,
	}
}

// startTelemetryHub manages active client broadcasting channels concurrently
func (hm *HandoverManager) startTelemetryHub() {
	for {
		select {
		case client := <-hm.registerWs:
			hm.mu.Lock()
			hm.wsClients[client] = true
			hm.mu.Unlock()
		case client := <-hm.unregisterWs:
			hm.mu.Lock()
			if _, ok := hm.wsClients[client]; ok {
				delete(hm.wsClients, client)
				close(client)
			}
			hm.mu.Unlock()
		}
	}
}

// BroadcastTelemetry sends real-time infrastructure event notifications to all WebSocket clients
func (hm *HandoverManager) BroadcastTelemetry(data interface{}) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	for client := range hm.wsClients {
		select {
		case client <- data:
		default:
			// Exception Handling: Drop slow or unresponsive streaming queues
		}
	}
}

// HandleRouterConnect handles new incoming connection requests with Geofence validation
func (hm *HandoverManager) HandleRouterConnect(deviceID, mac string, loc Location) (*Session, error) {
	if err := hm.billingManager.CheckGeofence(deviceID, loc); err != nil {
		return nil, fmt.Errorf("connection denied due to billing/geofence policy: %v", err)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	targetGW := "GW-HAN-01"
	if err := hm.gatewayPool.UpdateLoad(targetGW, 1); err != nil {
		return nil, err
	}

	sID := fmt.Sprintf("SES-%d", time.Now().UnixNano()%10000)
	session := &Session{
		ID:               sID,
		DeviceID:         deviceID,
		RouterMAC:        mac,
		SatelliteID:      hm.settings.DefaultSatelliteID,
		CurrentGatewayID: targetGW,
		State:            StateHold,
		StartTime:        time.Now(),
	}
	hm.sessions[sID] = session

	// Stream active link creation metrics over WebSockets
	go hm.BroadcastTelemetry(map[string]interface{}{
		"event":      "session_start",
		"session_id": sID,
		"device_id":  deviceID,
		"timestamp":  time.Now().Format(time.RFC3339),
	})

	return session, nil
}

// TriggerHandover simulates a handover process and logs historical tracking data
func (hm *HandoverManager) TriggerHandover(sessionID, targetGW string, currentLoc Location) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	session, exists := hm.sessions[sessionID]
	if !exists {
		return errors.New("session not found")
	}

	if err := hm.gatewayPool.UpdateLoad(targetGW, 0); err != nil {
        return errors.New("target gateway not found in registry")
    }

	startTime := time.Now()
	oldGW := session.CurrentGatewayID

	// Verification: Check Geofence during Handover Phase
	if err := hm.billingManager.CheckGeofence(session.DeviceID, currentLoc); err != nil {
		session.State = StateRelease
		hm.gatewayPool.UpdateLoad(session.CurrentGatewayID, -1)

		// Record the handover failure state into auditing archives
		hm.handoverHistory = append(hm.handoverHistory, HandoverEvent{
			ID:              fmt.Sprintf("HO-%d", time.Now().UnixNano()%10000),
			SessionID:       sessionID,
			SourceGatewayID: oldGW,
			TargetGatewayID: targetGW,
			DurationMs:      time.Since(startTime).Milliseconds(),
			Status:          "failed",
			Timestamp:       time.Now(),
		})
		return fmt.Errorf("handover aborted and session released: %v", err)
	}

	session.State = StatePrepare
	session.NextGatewayID = targetGW

	hm.gatewayPool.UpdateLoad(session.CurrentGatewayID, -1)
	hm.gatewayPool.UpdateLoad(targetGW, 1)

	session.CurrentGatewayID = targetGW
	session.NextGatewayID = ""
	session.State = StateHold

	// Append successful transmission to persistent records
	hoID := fmt.Sprintf("HO-%d", time.Now().UnixNano()%10000)
	duration := time.Since(startTime).Milliseconds()
	hm.handoverHistory = append(hm.handoverHistory, HandoverEvent{
		ID:              hoID,
		SessionID:       sessionID,
		SourceGatewayID: oldGW,
		TargetGatewayID: targetGW,
		DurationMs:      duration,
		Status:          "success",
		Timestamp:       time.Now(),
	})

	// Fire real-time data frame notification across pipelines
	go hm.BroadcastTelemetry(map[string]interface{}{
		"event":       "handover_execute",
		"handover_id": hoID,
		"duration_ms": duration,
		"status":      "success",
	})

	return nil
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

// GetHandoverHistory retrieves historical records filtering through exception windows safely
func (hm *HandoverManager) GetHandoverHistory() []HandoverEvent {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	res := make([]HandoverEvent, len(hm.handoverHistory))
	copy(res, hm.handoverHistory)
	return res
}
