package main

import "time"

type TelemetryEngine struct {
	handoverManager *HandoverManager
	gatewayPool     *GatewayPool
	satelliteTracker *SatelliteTracker
	rateHz          int
	stopCh          chan struct{}
}

func NewTelemetryEngine(hm *HandoverManager, pool *GatewayPool, tracker *SatelliteTracker, rateHz int) *TelemetryEngine {
	if rateHz <= 0 {
		rateHz = 5
	}
	return &TelemetryEngine{
		handoverManager: hm,
		gatewayPool:      pool,
		satelliteTracker: tracker,
		rateHz:           rateHz,
		stopCh:           make(chan struct{}),
	}
}

func (te *TelemetryEngine) Start() {
	go te.loop()
}

func (te *TelemetryEngine) Stop() {
	close(te.stopCh)
}

func (te *TelemetryEngine) loop() {
	interval := time.Second / time.Duration(te.rateHz)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now().UTC()
			te.satelliteTracker.Update(now)
			satStates := te.satelliteTracker.GetStates()
			dtSeconds := 1.0 / float64(te.rateHz)
			telemetry := te.handoverManager.UpdateSessions(now, satStates, dtSeconds)

			snapshot := TelemetrySnapshot{
				Gateways:  te.gatewayPool.GetAllGateways(),
				Sessions:  te.handoverManager.GetActiveSessions(),
				Handovers: te.handoverManager.GetHandoverHistory(),
				Satellites: satStates,
				Telemetry: telemetry,
				Timestamp: now,
			}
			te.handoverManager.BroadcastSnapshot(snapshot)
		case <-te.stopCh:
			return
		}
	}
}
