package main

import "time"

// DeployConfig holds environment-specific paths and port
type DeployConfig struct {
	AppPort            string `json:"app_port"`
	GatewayDataPath    string `json:"gateway_data_path"`
	SystemSettingsPath string `json:"settings_path"`
	TlePath            string `json:"tle_path"`
}

// SystemSettings holds configurable parameters from settings.json
type SystemSettings struct {
	EarthRadiusKm       float64 `json:"earth_radius_km"`       // Mean radius of Earth
	ElevationHysteresis float64 `json:"elevation_hysteresis"`  // Safe margin for handover
	DefaultSatelliteID  string  `json:"default_satellite_id"`  // Default SAT ID for mock
	TelemetryBufferSize int     `json:"telemetry_buffer_size"` // Channel capacity
	ISPSecretSalt       string  `json:"isp_secret_salt"`
	AuditLogPath        string  `json:"audit_log_path"`
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
	DeviceID         string       `json:"device_id"`
	RouterMAC        string       `json:"router_mac"`
	SatelliteID      string       `json:"satellite_id"`
	CurrentGatewayID string       `json:"current_gateway_id"`
	NextGatewayID    string       `json:"next_gateway_id,omitempty"`
	State            SessionState `json:"state"`
	StartTime        time.Time    `json:"start_time"`
	Location         Location     `json:"location"`
	CnRatioDb        float64      `json:"cn_ratio_db,omitempty"`
	DataDownMbps     float64      `json:"data_down_mbps,omitempty"`
	DataUpMbps       float64      `json:"data_up_mbps,omitempty"`
	DataMb           float64      `json:"data_mb,omitempty"`
	PacketLossPct    float64      `json:"packet_loss_pct,omitempty"`
	LatencyMs        float64      `json:"latency_ms,omitempty"`
	JitterMs         float64      `json:"jitter_ms,omitempty"`
	LinkStatus       string       `json:"link_status,omitempty"`
	LastActivity     time.Time    `json:"last_activity,omitempty"`
	DeviceStatus     DeviceStatus `json:"device_status,omitempty"`
	HandoverActive   bool         `json:"handover_active,omitempty"`
	LastHandoverAt   time.Time    `json:"last_handover_at,omitempty"`
}

// TelemetryEvent for real-time monitoring broadcasting
type TelemetryEvent struct {
	Timestamp   time.Time   `json:"timestamp"`
	EventType   string      `json:"event_type"`
	GatewayID   string      `json:"gateway_id"`
	Metrics     interface{} `json:"metrics"`
	Description string      `json:"description"`
}

// TelemetrySnapshot contains a full snapshot for WebSocket subscribers
type TelemetrySnapshot struct {
	Gateways   []Gateway         `json:"gateways,omitempty"`
	Sessions   []Session         `json:"sessions,omitempty"`
	Handovers  []HandoverEvent   `json:"handovers,omitempty"`
	Satellites []Satellite       `json:"satellites,omitempty"`
	Telemetry  []SessionTelemetry `json:"telemetry,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// --- ADDED FOR PHASE 1 EXTENSION: SATELLITES & HANDOVERS ---

// Satellite represents the state and orbit information of a LEO satellite
type Satellite struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Location  Location  `json:"location"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Altitude  float64   `json:"altitude"`
	Elevation float64   `json:"elevation"`
	Azimuth   float64   `json:"azimuth"`
	Timestamp time.Time `json:"timestamp"`
}

// HandoverEvent tracks historical handover transitions between gateways
type HandoverEvent struct {
	ID              string    `json:"id"`
	SessionID       string    `json:"session_id"`
	SourceGatewayID string    `json:"source_gateway_id"`
	TargetGatewayID string    `json:"target_gateway_id"`
	DurationMs      int64     `json:"duration_ms"`
	Status          string    `json:"status"` // "success" or "failed"
	PacketLossPct   float64   `json:"packet_loss_pct,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// SessionTelemetry provides rich per-session metrics for client UI streaming
type SessionTelemetry struct {
	DeviceID        string       `json:"device_id"`
	SessionID       string       `json:"session_id"`
	GatewayID       string       `json:"gateway_id"`
	GatewayName     string       `json:"gateway_name"`
	SatelliteID     string       `json:"satellite_id"`
	SatelliteName   string       `json:"satellite_name"`
	Location        Location     `json:"location"`
	AzimuthDeg      float64      `json:"azimuth_deg"`
	ElevationDeg    float64      `json:"elevation_deg"`
	RangeKm         float64      `json:"range_km"`
	BeamQuality     float64      `json:"beam_quality"`
	CarrierPowerDbm float64      `json:"carrier_power_dbm"`
	CnRatioDb       float64      `json:"c_n_ratio_db"`
	EbN0Db          float64      `json:"eb_n0_db"`
	Ber             float64      `json:"ber"`
	PathLossDb      float64      `json:"path_loss_db"`
	EirpDbw         float64      `json:"eirp_dbw"`
	Modulation      string       `json:"modulation_scheme"`
	LinkStatus      string       `json:"link_status"`
	TimeToHorizonS  int          `json:"time_to_horizon_s"`
	HandoverActive  bool         `json:"handover_active"`
	PacketLossPct   float64      `json:"packet_loss_pct"`
	LatencyMs       float64      `json:"latency_ms"`
	JitterMs        float64      `json:"jitter_ms"`
	DataDownMbps    float64      `json:"data_down_mbps"`
	DataUpMbps      float64      `json:"data_up_mbps"`
	DeviceStatus    DeviceStatus `json:"device_status"`
}

// --- ADDED FOR PHASE 2: DEVICE PROVISIONING ---

// DeviceStatus defines the lifecycle of a router hardware
type DeviceStatus string

const (
	DeviceRegistered DeviceStatus = "registered"
	DeviceActive     DeviceStatus = "active"
	DeviceSuspended  DeviceStatus = "suspended"
	DeviceRevoked    DeviceStatus = "revoked"
)

// DeviceRecord holds hardware identity and security status
type DeviceRecord struct {
	DeviceID          string       `json:"device_id"`
	MACAddress        string       `json:"mac"`
	HardwareID        string       `json:"hw_id"`
	Model             string       `json:"model,omitempty"`
	Owner             string       `json:"owner,omitempty"`
	Status            DeviceStatus `json:"status"`
	ProvisioningToken string       `json:"provisioning_token"`
	RegisteredAt      time.Time    `json:"registered_at"`
	RevokedAt         *time.Time   `json:"revoked_at,omitempty"`
}

// RegisterDeviceReq represents the payload for POST /api/devices/register
type RegisterDeviceReq struct {
	DeviceID string `json:"device_id" binding:"required"`
	MAC      string `json:"mac" binding:"required"`
	HWID     string `json:"hw_id" binding:"required"`
	Model    string `json:"model"`
}

// VerifyDeviceReq represents the payload for POST /api/devices/verify
type VerifyDeviceReq struct {
	DeviceID  string `json:"device_id" binding:"required"`
	CSRPEM    string `json:"csr_pem" binding:"required"`
	Nonce     string `json:"nonce" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

// RevokeDeviceReq represents the payload for POST /api/devices/revoke
type RevokeDeviceReq struct {
	DeviceID string `json:"device_id" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

// --- ADDED FOR PHASE 4: SPATIOTEMPORAL BILLING ---

// PlanType defines the subscription business model
type PlanType string

const (
	PlanFixed  PlanType = "fixed"  // Geofenced to a specific radius (e.g., 50km)
	PlanMobile PlanType = "mobile" // Roaming allowed anywhere, billed by usage
)

// SubscriptionRecord holds billing and geofencing rules for a specific device
type SubscriptionRecord struct {
	DeviceID        string    `json:"device_id"`
	Plan            PlanType  `json:"plan"`
	HomeLocation    *Location `json:"home_location,omitempty"` // Required if Plan == PlanFixed
	AllowedRadiusKm float64   `json:"allowed_radius_km"`       // Max drift distance (e.g., 50.0)
}

// GeofenceOverride defines a temporary bypass for geofence restrictions (e.g., emergencies)
type GeofenceOverride struct {
	OverrideID string    `json:"override_id"`
	DeviceID   string    `json:"device_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	Reason     string    `json:"reason"`
}

// BillingEventRecord tracks network usage for ISP accounting
type BillingEventRecord struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"type"` // "start", "stop", "usage"
	SessionID  string    `json:"session_id"`
	DeviceID   string    `json:"device_id"`
	BytesDelta int64     `json:"bytes_delta"`
	Timestamp  time.Time `json:"ts"`
}

// --- API Request Payloads (Matching api.md) ---

// GeofenceOverrideReq represents payload for POST /api/geofence/override
type GeofenceOverrideReq struct {
	DeviceID string `json:"device_id" binding:"required"`
	TTLS     int    `json:"ttl_s" binding:"required,min=1"`
	Reason   string `json:"reason" binding:"required"`
}

// BillingEventReq represents payload for POST /api/billing/event
type BillingEventReq struct {
	EventID    string    `json:"event_id" binding:"required"`
	Type       string    `json:"type" binding:"required"`
	SessionID  string    `json:"session_id" binding:"required"`
	DeviceID   string    `json:"device_id" binding:"required"`
	BytesDelta int64     `json:"bytes_delta"`
	Timestamp  time.Time `json:"ts" binding:"required"`
}
