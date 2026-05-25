package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

// GatewayPool manages the ground station network
type GatewayPool struct {
	gateways    map[string]*Gateway
	settings    SystemSettings
	mu          sync.RWMutex
	TelemetryCh chan TelemetryEvent
}

// NewGatewayPool initializes the pool with provided settings
func NewGatewayPool(settings SystemSettings) *GatewayPool {
	return &GatewayPool{
		gateways:    make(map[string]*Gateway),
		settings:    settings,
		TelemetryCh: make(chan TelemetryEvent, settings.TelemetryBufferSize),
	}
}

// AddGateway adds a gateway to the registry
func (p *GatewayPool) AddGateway(g *Gateway) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gateways[g.ID] = g
}

// GetAllGateways returns a list of all gateways
func (p *GatewayPool) GetAllGateways() []Gateway {
	p.mu.RLock()
	defer p.mu.RUnlock()
	list := make([]Gateway, 0, len(p.gateways))
	for _, g := range p.gateways {
		list = append(list, *g)
	}
	return list
}

// CalculateElevation performs core geometric calculations for link viability
func (p *GatewayPool) CalculateElevation(gwID string, satLoc Location) (float64, float64, error) {
	p.mu.RLock()
	g, exists := p.gateways[gwID]
	p.mu.RUnlock()
	if !exists {
		return 0, 0, fmt.Errorf("gateway %s not found", gwID)
	}

	lat1, lon1 := g.Location.Latitude*math.Pi/180, g.Location.Longitude*math.Pi/180
	lat2, lon2 := satLoc.Latitude*math.Pi/180, satLoc.Longitude*math.Pi/180

	dLon, dLat := lon2-lon1, lat2-lat1
	a := math.Pow(math.Sin(dLat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dLon/2), 2)
	if a > 1.0 {
		a = 1.0
	}
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	rGate := p.settings.EarthRadiusKm + g.Location.Altitude
	rSat := p.settings.EarthRadiusKm + satLoc.Altitude

	distanceKm := math.Sqrt(math.Pow(rGate, 2) + math.Pow(rSat, 2) - 2*rGate*rSat*math.Cos(c))
	sinElev := (rSat*math.Cos(c) - rGate) / distanceKm
	if sinElev > 1.0 {
		sinElev = 1.0
	} else if sinElev < -1.0 {
		sinElev = -1.0
	}

	return math.Asin(sinElev) * 180 / math.Pi, distanceKm, nil
}

// UpdateLoad increments or decrements session counters
func (p *GatewayPool) UpdateLoad(gwID string, delta int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	g, ok := p.gateways[gwID]
	if !ok || g.Status != "alive" {
		return fmt.Errorf("gw unavailable")
	}

	newVal := g.CurrentSessions + delta
	if newVal < 0 || newVal > g.MaxSessions {
		return fmt.Errorf("capacity error")
	}
	g.CurrentSessions = newVal
	return nil
}

// LogTelemetry dispatches events to terminal and telemetry channel
func (p *GatewayPool) LogTelemetry(eType, gwID string, metrics interface{}, desc string) {
	event := TelemetryEvent{Timestamp: time.Now(), EventType: eType, GatewayID: gwID, Metrics: metrics, Description: desc}
	raw, _ := json.Marshal(event)
	fmt.Println(string(raw))
	select {
	case p.TelemetryCh <- event:
	default:
	}
}
