package main

import "time"

// DeployConfig holds environment-specific paths and port
type DeployConfig struct {
	AppPort            string `json:"app_port"`
	GatewayDataPath    string `json:"gateway_data_path"`
	SystemSettingsPath string `json:"settings_path"`
}

// SystemSettings holds configurable parameters from settings.json
type SystemSettings struct {
	EarthRadiusKm       float64 `json:"earth_radius_km"`       // Mean radius of Earth
	ElevationHysteresis float64 `json:"elevation_hysteresis"`  // Safe margin for handover
	DefaultSatelliteID  string  `json:"default_satellite_id"`  // Default SAT ID for mock
	TelemetryBufferSize int     `json:"telemetry_buffer_size"` // Channel capacity
}

// SessionState defines the lifecycle phases of a connection
type SessionState string

const (
	StateAcquire SessionState = "Acquire" // Election phase
	StateHold    SessionState = "Hold"    // Connected phase
	StatePrepare SessionState = "Prepare" // Pre-handover phase
	StateExecute SessionState = "Execute" // Switching phase
	StateRelease SessionState = "Release" // Cleanup phase
)

// Location stores 3D geographic coordinates
type Location struct {
	Latitude  float64 `json:"lat"` // Decimal degrees
	Longitude float64 `json:"lon"` // Decimal degrees
	Altitude  float64 `json:"alt"` // Altitude in km
}

// AntennaModel describes radio hardware specs
type AntennaModel struct {
	GainDBi      float64 `json:"gain_dbi"`
	BeamWidthDeg float64 `json:"beam_width_deg"`
}

// Gateway represents a ground station infrastructure
type Gateway struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Location        Location     `json:"location"`
	MaxSessions     int          `json:"max_sessions"`
	CurrentSessions int          `json:"current_sessions"`
	MinElevationDeg float64      `json:"min_elevation_deg"`
	Antenna         AntennaModel `json:"antenna"`
	Status          string       `json:"status"` // alive, dead, degraded
}

// Session represents an active router connection
type Session struct {
	ID               string       `json:"session_id"`
	RouterMAC        string       `json:"router_mac"`
	SatelliteID      string       `json:"satellite_id"`
	CurrentGatewayID string       `json:"current_gateway_id"`
	NextGatewayID    string       `json:"next_gateway_id,omitempty"`
	State            SessionState `json:"state"`
	StartTime        time.Time    `json:"start_time"`
}

// TelemetryEvent for real-time monitoring broadcasting
type TelemetryEvent struct {
	Timestamp   time.Time   `json:"timestamp"`
	EventType   string      `json:"event_type"`
	GatewayID   string      `json:"gateway_id"`
	Metrics     interface{} `json:"metrics"`
	Description string      `json:"description"`
}
