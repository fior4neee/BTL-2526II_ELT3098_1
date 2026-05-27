package main

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// OrbitUpdate represents data from orbit_calc module
type OrbitUpdate struct {
	SatelliteID string
	Location    Location
}

type wsClient struct {
	ch     chan interface{}
	topics map[string]bool
}

// HandoverManager handles connection FSM, transitions, and historic tracking
type HandoverManager struct {
	sessions        map[string]*Session
	gatewayPool     *GatewayPool
	settings        SystemSettings
	billingManager  *BillingManager
	provisioning    *ProvisioningManager
	satTracker      *SatelliteTracker
	handoverHistory []HandoverEvent
	wsClients       map[chan interface{}]wsClient
	registerWs      chan wsClient
	unregisterWs    chan chan interface{}
	mu              sync.RWMutex
}

// NewHandoverManager initializes manager with system settings, billing engine, and live telemetry channels
func NewHandoverManager(pool *GatewayPool, settings SystemSettings, bm *BillingManager, pm *ProvisioningManager, st *SatelliteTracker) *HandoverManager {
	hm := &HandoverManager{
		sessions:        make(map[string]*Session),
		gatewayPool:     pool,
		settings:        settings,
		billingManager:  bm,
		provisioning:    pm,
		satTracker:      st,
		handoverHistory: make([]HandoverEvent, 0),
		wsClients:       make(map[chan interface{}]wsClient),
		registerWs:      make(chan wsClient),
		unregisterWs:    make(chan chan interface{}),
	}

	// Start an asynchronous central hub loop to manage active WebSocket listeners safely
	go hm.startTelemetryHub()
	return hm
}

// startTelemetryHub manages active client broadcasting channels concurrently
func (hm *HandoverManager) startTelemetryHub() {
	for {
		select {
		case client := <-hm.registerWs:
			hm.mu.Lock()
			hm.wsClients[client.ch] = client
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

// BroadcastSnapshot sends real-time telemetry snapshots to all WebSocket clients
func (hm *HandoverManager) BroadcastSnapshot(snapshot TelemetrySnapshot) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	for _, client := range hm.wsClients {
		payload := map[string]interface{}{
			"timestamp": snapshot.Timestamp,
		}
		if client.topics["gateways"] {
			payload["gateways"] = snapshot.Gateways
		}
		if client.topics["sessions"] {
			payload["sessions"] = snapshot.Sessions
		}
		if client.topics["handovers"] {
			payload["handovers"] = snapshot.Handovers
		}
		if client.topics["satellites"] {
			payload["satellites"] = snapshot.Satellites
		}
		if client.topics["telemetry"] {
			payload["telemetry"] = snapshot.Telemetry
		}

		select {
		case client.ch <- payload:
		default:
			// Drop slow or unresponsive streaming queues
		}
	}
}

// HandleRouterConnect handles new incoming connection requests with Geofence validation
func (hm *HandoverManager) HandleRouterConnect(deviceID, mac string, loc Location) (*Session, error) {
	if hm.provisioning != nil {
		if status, ok := hm.provisioning.GetDeviceStatus(deviceID); ok {
			if status == DeviceSuspended || status == DeviceRevoked {
				return nil, fmt.Errorf("device status %s", status)
			}
		}
	}
	if err := hm.billingManager.CheckGeofence(deviceID, loc); err != nil {
		return nil, fmt.Errorf("connection denied due to billing/geofence policy: %v", err)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	sats := hm.satTracker.GetStates()
	if len(sats) == 0 {
		hm.satTracker.Update(time.Now().UTC())
		sats = hm.satTracker.GetStates()
	}

	best := hm.selectBestLink(loc, sats)
	if best == nil {
		return nil, fmt.Errorf("no viable satellite/gateway link")
	}

	if err := hm.gatewayPool.UpdateLoad(best.Gateway.ID, 1); err != nil {
		return nil, err
	}

	sID := fmt.Sprintf("SES-%d", time.Now().UnixNano()%10000)
	session := &Session{
		ID:               sID,
		DeviceID:         deviceID,
		RouterMAC:        mac,
		SatelliteID:      best.Satellite.ID,
		CurrentGatewayID: best.Gateway.ID,
		State:            StateHold,
		StartTime:        time.Now(),
		Location:         loc,
		LinkStatus:       "connected",
		LastActivity:     time.Now(),
	}
	if hm.provisioning != nil {
		if status, ok := hm.provisioning.GetDeviceStatus(deviceID); ok {
			session.DeviceStatus = status
		} else {
			session.DeviceStatus = DeviceRegistered
		}
	}
	hm.sessions[sID] = session

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
			PacketLossPct:   0.3,
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
	session.State = StateExecute
	session.LastHandoverAt = time.Now()
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
		PacketLossPct:   0.02,
		Timestamp:       time.Now(),
	})

	return nil
}

type linkCandidate struct {
	Satellite  Satellite
	Gateway    *Gateway
	AzimuthDeg float64
	ElevDeg    float64
	RangeKm    float64
	Score      float64
}

func (hm *HandoverManager) selectBestLink(loc Location, sats []Satellite) *linkCandidate {
	if len(sats) == 0 {
		return nil
	}
	gateways := hm.gatewayPool.GetAllGateways()
	var best *linkCandidate
	for _, sat := range sats {
		az, elev, rng := calcLookAngles(loc, sat.Location)
		if elev < 15.0 {
			continue
		}
		for _, gw := range gateways {
			if gw.Status != "alive" {
				continue
			}
			gwElev, _, err := hm.gatewayPool.CalculateElevation(gw.ID, sat.Location)
			if err != nil {
				continue
			}
			minElev := gw.MinElevationDeg
			if minElev <= 0 {
				minElev = 15
			}
			if gwElev < minElev {
				continue
			}
			distKm := hm.billingManager.calculateDistance(loc, gw.Location)
			score := elev + (gwElev * 0.5) - (distKm * 0.1)
			gwCopy := gw
			cand := &linkCandidate{
				Satellite:  sat,
				Gateway:    &gwCopy,
				AzimuthDeg: az,
				ElevDeg:    elev,
				RangeKm:    rng,
				Score:      score,
			}
			if best == nil || cand.Score > best.Score {
				best = cand
			}
		}
	}
	return best
}

// UpdateSessions recalculates link state, performs handovers, and returns telemetry samples
func (hm *HandoverManager) UpdateSessions(now time.Time, sats []Satellite, dtSeconds float64) []SessionTelemetry {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	telemetry := make([]SessionTelemetry, 0, len(hm.sessions))
	for _, session := range hm.sessions {
		status, exists := hm.provisioning.GetDeviceStatus(session.DeviceID)
		isBlocked := !exists || status == DeviceSuspended || status == DeviceRevoked

		best := hm.selectBestLink(session.Location, sats)
		if best == nil || isBlocked {
			session.LinkStatus = "outage"
			session.DataDownMbps = 0
			session.DataUpMbps = 0
			session.PacketLossPct = 1.0
			session.LatencyMs = 999
			session.JitterMs = 99
			session.LastActivity = now
			telemetry = append(telemetry, hm.buildTelemetry(session, nil, 0, 0, 0, 0, 0, 0, 0))
			continue
		}

		currentScore := -1.0
		if session.SatelliteID != "" && session.CurrentGatewayID != "" {
			for _, sat := range sats {
				if sat.ID != session.SatelliteID {
					continue
				}
				az, elev, _ := calcLookAngles(session.Location, sat.Location)
				gwElev, _, err := hm.gatewayPool.CalculateElevation(session.CurrentGatewayID, sat.Location)
				if err != nil {
					continue
				}
				currentScore = elev + (gwElev * 0.5)
				_ = az
				break
			}
		}

		hysteresis := hm.settings.ElevationHysteresis
		if hysteresis <= 0 {
			hysteresis = 2
		}

		needsHandover := session.CurrentGatewayID != best.Gateway.ID || session.SatelliteID != best.Satellite.ID
		if needsHandover && (currentScore < 0 || best.Score-currentScore > hysteresis) {
			oldGW := session.CurrentGatewayID
			start := time.Now()
			hm.gatewayPool.UpdateLoad(oldGW, -1)
			hm.gatewayPool.UpdateLoad(best.Gateway.ID, 1)
			session.State = StateExecute
			session.CurrentGatewayID = best.Gateway.ID
			session.SatelliteID = best.Satellite.ID
			session.LastHandoverAt = now
			session.State = StateHold

			hm.handoverHistory = append(hm.handoverHistory, HandoverEvent{
				ID:              fmt.Sprintf("HO-%d", time.Now().UnixNano()%10000),
				SessionID:       session.ID,
				SourceGatewayID: oldGW,
				TargetGatewayID: best.Gateway.ID,
				DurationMs:      time.Since(start).Milliseconds(),
				Status:          "success",
				PacketLossPct:   0.02,
				Timestamp:       now,
			})
		}

		cnDb, ebN0Db, pathLossDb, carrierPowerDbm := calcLinkMetrics(best.RangeKm, best.ElevDeg)
		ber := qpskBer(ebN0Db)
		linkStatus := "connected"
		if best.ElevDeg < 15 || cnDb < 9 {
			linkStatus = "searching"
		}
		if best.ElevDeg < 8 || cnDb < 6 {
			linkStatus = "outage"
		}

		speedDown := 0.0
		speedUp := 0.0
		if linkStatus == "connected" {
			speedDown = clamp(20+cnDb*1.8, 2, 120)
			speedUp = clamp(4+cnDb*0.5, 1, 35)
		}

		deltaMb := (speedDown+speedUp)*1_000_000/8/1024/1024*dtSeconds
		session.DataMb += deltaMb
		session.CnRatioDb = cnDb
		session.DataDownMbps = speedDown
		session.DataUpMbps = speedUp
		session.PacketLossPct = clamp(0.01+math.Max(0, 13-cnDb)*0.01, 0.01, 0.2)
		session.LatencyMs = clamp(35+math.Max(0, 18-cnDb)*2.4, 25, 180)
		session.JitterMs = clamp(2+math.Max(0, 16-cnDb)*0.7, 1.5, 25)
		session.LinkStatus = linkStatus
		session.LastActivity = now
		session.HandoverActive = now.Sub(session.LastHandoverAt) < 2*time.Second

		telemetry = append(telemetry, hm.buildTelemetry(session, best, cnDb, ebN0Db, pathLossDb, carrierPowerDbm, ber, speedDown, speedUp))
	}

	return telemetry
}

func (hm *HandoverManager) buildTelemetry(session *Session, best *linkCandidate, cnDb, ebN0Db, pathLossDb, carrierPowerDbm, ber, speedDown, speedUp float64) SessionTelemetry {
	telemetry := SessionTelemetry{
		DeviceID:     session.DeviceID,
		SessionID:    session.ID,
		GatewayID:    session.CurrentGatewayID,
		GatewayName:  session.CurrentGatewayID,
		SatelliteID:  session.SatelliteID,
		SatelliteName: session.SatelliteID,
		Location:     session.Location,
		CnRatioDb:    cnDb,
		EbN0Db:       ebN0Db,
		Ber:          ber,
		PathLossDb:   pathLossDb,
		CarrierPowerDbm: carrierPowerDbm,
		EirpDbw:      defaultEirpDbw,
		Modulation:   modulationForCn(cnDb),
		LinkStatus:   session.LinkStatus,
		HandoverActive: session.HandoverActive,
		PacketLossPct: session.PacketLossPct,
		LatencyMs:    session.LatencyMs,
		JitterMs:     session.JitterMs,
		DataDownMbps: speedDown,
		DataUpMbps:   speedUp,
		DeviceStatus: session.DeviceStatus,
	}
	if best != nil {
		telemetry.GatewayID = best.Gateway.ID
		telemetry.GatewayName = best.Gateway.Name
		telemetry.SatelliteID = best.Satellite.ID
		telemetry.SatelliteName = best.Satellite.Name
		telemetry.AzimuthDeg = best.AzimuthDeg
		telemetry.ElevationDeg = best.ElevDeg
		telemetry.RangeKm = best.RangeKm
		telemetry.BeamQuality = clamp(0.48+(best.ElevDeg/90)*0.47, 0.25, 0.99)
		telemetry.TimeToHorizonS = int(clamp((best.ElevDeg-4)*22, 0, 1800))
	}

	if hm.provisioning != nil {
		if status, ok := hm.provisioning.GetDeviceStatus(session.DeviceID); ok {
			session.DeviceStatus = status
			telemetry.DeviceStatus = status
		} else {
			session.DeviceStatus = DeviceRegistered
			telemetry.DeviceStatus = DeviceRegistered
		}
	}

	return telemetry
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

// UpdateDeviceLocation updates session location for a device if session exists
func (hm *HandoverManager) UpdateDeviceLocation(deviceID string, loc Location) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	for _, s := range hm.sessions {
		if s.DeviceID == deviceID {
			s.Location = loc
			s.LastActivity = time.Now()
			return nil
		}
	}
	return fmt.Errorf("session not found for device")
}
