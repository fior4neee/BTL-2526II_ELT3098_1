package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// GatewayPool is the central manager for ground station infrastructure
type GatewayPool struct {
	gateways    map[string]*Gateway
	mu          sync.RWMutex
	TelemetryCh chan TelemetryEvent // Broadcast channel for WebSocket server
}

// NewGatewayPool initializes the pool with an empty registry and telemetry buffer
func NewGatewayPool() *GatewayPool {
	return &GatewayPool{
		gateways:    make(map[string]*Gateway),
		TelemetryCh: make(chan TelemetryEvent, 1000),
	}
}

// AddGateway registers a new gateway into the system
func (p *GatewayPool) AddGateway(g *Gateway) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gateways[g.ID] = g
}

// Returns a snapshot of all gateways for API responses
func (p *GatewayPool) GetAllGateways() []Gateway {
	p.mu.RLock()
	defer p.mu.RUnlock()
	list := make([]Gateway, 0, len(p.gateways))
	for _, g := range p.gateways {
		list = append(list, *g)
	}
	return list
}

// GetGateway retrieves a specific gateway by its ID
func (p *GatewayPool) GetGateway(id string) (*Gateway, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	g, exists := p.gateways[id]
	if !exists {
		return nil, fmt.Errorf("gateway %s not found", id)
	}
	return g, nil
}

// CalculateElevation computes elevation angle and slant range using spherical trigonometry
func (p *GatewayPool) CalculateElevation(gwID string, satLoc Location) (float64, float64, error) {
	g, err := p.GetGateway(gwID)
	if err != nil {
		return 0, 0, err
	}

	if satLoc.Altitude < 0 {
		return 0, 0, errors.New("invalid satellite altitude")
	}

	// Degree to Radian conversion
	lat1 := g.Location.Latitude * math.Pi / 180
	lon1 := g.Location.Longitude * math.Pi / 180
	lat2 := satLoc.Latitude * math.Pi / 180
	lon2 := satLoc.Longitude * math.Pi / 180

	// Calculate central angle 'c' using Haversine formula
	dLon := lon2 - lon1
	dLat := lat2 - lat1
	a := math.Pow(math.Sin(dLat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dLon/2), 2)

	if a > 1.0 {
		a = 1.0
	} else if a < 0.0 {
		a = 0.0
	}
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	rGate := EarthRadiusKm + g.Location.Altitude
	rSat := EarthRadiusKm + satLoc.Altitude

	// Law of Cosines to find the Slant Range (Distance)
	distanceKm := math.Sqrt(math.Pow(rGate, 2) + math.Pow(rSat, 2) - 2*rGate*rSat*math.Cos(c))

	// Find Sin of Elevation angle
	sinElevation := (rSat*math.Cos(c) - rGate) / distanceKm

	if sinElevation > 1.0 {
		sinElevation = 1.0
	} else if sinElevation < -1.0 {
		sinElevation = -1.0
	}

	elevationDeg := math.Asin(sinElevation) * 180 / math.Pi

	return elevationDeg, distanceKm, nil
}

// UpdateLoad adjusts the session count for a gateway with capacity checks
func (p *GatewayPool) UpdateLoad(gwID string, delta int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	g, exists := p.gateways[gwID]
	if !exists {
		return errors.New("gateway not found")
	}
	if g.Status != "alive" {
		return errors.New("gateway is offline")
	}

	newLoad := g.CurrentSessions + delta
	if newLoad < 0 {
		newLoad = 0
	}
	if newLoad > g.MaxSessions {
		return fmt.Errorf("capacity limit exceeded for %s", gwID)
	}

	g.CurrentSessions = newLoad
	return nil
}

// LogTelemetry emits events to stdout and internal channel for real-time monitoring
func (p *GatewayPool) LogTelemetry(eventType, gatewayID string, metrics interface{}, desc string) {
	event := TelemetryEvent{
		Timestamp:   time.Now(),
		EventType:   eventType,
		GatewayID:   gatewayID,
		Metrics:     metrics,
		Description: desc,
	}

	logBytes, _ := json.Marshal(event)
	fmt.Println(string(logBytes))

	select {
	case p.TelemetryCh <- event:
	default: // Non-blocking drop if channel is full
	}
}
