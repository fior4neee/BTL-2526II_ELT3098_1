package main

import "time"

// Physical constants and system settings
const (
	EarthRadiusKm       = 6371.0 // Mean radius of the Earth in km
	ElevationHysteresis = 2.0    // Dead-zone (degrees) to prevent ping-pong effect during handover
)

// SessionState defines the possible lifecycle phases of a connection
type SessionState string

const (
	StateAcquire SessionState = "Acquire" // Initial connection and gateway election
	StateHold    SessionState = "Hold"    // Stable connection phase
	StatePrepare SessionState = "Prepare" // Approaching horizon, predicting next gateway
	StateExecute SessionState = "Execute" // Executing sub-100ms switch
	StateRelease SessionState = "Release" // Cleaning up old gateway resources
)

// Location stores 3D geographic coordinates
type Location struct {
	Latitude  float64 `json:"lat"` // Latitude in decimal degrees
	Longitude float64 `json:"lon"` // Longitude in decimal degrees
	Altitude  float64 `json:"alt"` // Altitude in kilometers
}

// AntennaModel describes technical specifications of the ground station antenna
type AntennaModel struct {
	GainDBi      float64 `json:"gain_dbi"`       // Antenna gain in dBi
	BeamWidthDeg float64 `json:"beam_width_deg"` // Beam width in degrees
}

// Gateway represents a ground station infrastructure
type Gateway struct {
	ID              string       `json:"id"`                // Unique identifier for the gateway
	Name            string       `json:"name"`              // Human-readable station name
	Location        Location     `json:"location"`          // Physical location of the station
	MaxSessions     int          `json:"max_sessions"`      // Max concurrent session capacity
	CurrentSessions int          `json:"current_sessions"`  // Current number of active sessions
	MinElevationDeg float64      `json:"min_elevation_deg"` // Min angle to maintain link viability
	Antenna         AntennaModel `json:"antenna"`           // Radio hardware details
	Status          string       `json:"status"`            // Current health: alive, degraded, dead
	RecentHandovers int          `json:"recent_handovers"`  // Total successful handovers executed
}

// Session represents an active end-user router connection
type Session struct {
	ID               string       `json:"session_id"`                // Unique session UUID
	RouterMAC        string       `json:"router_mac"`                // Hardware MAC of the client router
	SatelliteID      string       `json:"satellite_id"`              // ID of the serving LEO satellite
	CurrentGatewayID string       `json:"current_gateway_id"`        // ID of the active ground station
	NextGatewayID    string       `json:"next_gateway_id,omitempty"` // Candidate for predictive handover
	State            SessionState `json:"state"`                     // Current FSM state
	CNRatio          float64      `json:"cn_ratio_db"`               // Carrier-to-Noise ratio in dB
	SNR              float64      `json:"snr_db"`                    // Signal-to-Noise ratio in dB
	PacketLoss       float64      `json:"packet_loss_pct"`           // Real-time packet loss percentage
	StartTime        time.Time    `json:"start_time"`                // Timestamp of session creation
}

// HandoverEvent stores the history of a single gateway-to-gateway migration
type HandoverEvent struct {
	EventID   string    `json:"ho_id"`      // Unique event identifier
	SessionID string    `json:"session_id"` // Associated session ID
	OldGW     string    `json:"old_gw"`     // Source gateway ID
	NewGW     string    `json:"new_gw"`     // Target gateway ID
	LossPct   float64   `json:"loss_pct"`   // Packet loss during the switch
	Timestamp time.Time `json:"time"`       // Execution timestamp
}

// TelemetryEvent used for broadcasting system logs via WebSocket (Global Firehose)
type TelemetryEvent struct {
	Timestamp   time.Time   `json:"timestamp"`   // Time of the event
	EventType   string      `json:"event_type"`  // e.g., HANDOVER, ALERT, CAPACITY
	GatewayID   string      `json:"gateway_id"`  // Associated gateway
	Metrics     interface{} `json:"metrics"`     // Dynamic metrics payload
	Description string      `json:"description"` // Human-readable log
}
