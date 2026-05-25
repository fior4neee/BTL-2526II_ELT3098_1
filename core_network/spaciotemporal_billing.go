package main

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BillingManager handles geofencing checks, plan enforcement, and usage tracking
type BillingManager struct {
	mu            sync.RWMutex
	subscriptions map[string]*SubscriptionRecord
	overrides     map[string]*GeofenceOverride
	billingEvents map[string]*BillingEventRecord
	earthRadius   float64
}

// NewBillingManager initializes the Spatiotemporal Billing engine
func NewBillingManager(settings SystemSettings) *BillingManager {
	radius := settings.EarthRadiusKm
	// Exception Handling: Fallback if Earth radius is improperly configured
	if radius <= 0 {
		radius = 6371.0 // Standard Earth radius in km
	}

	return &BillingManager{
		subscriptions: make(map[string]*SubscriptionRecord),
		overrides:     make(map[string]*GeofenceOverride),
		billingEvents: make(map[string]*BillingEventRecord),
		earthRadius:   radius,
	}
}

// calculateDistance uses the Haversine formula to compute the great-circle distance between two points
func (bm *BillingManager) calculateDistance(loc1, loc2 Location) float64 {
	// Convert decimal degrees to radians
	lat1 := loc1.Latitude * math.Pi / 180
	lon1 := loc1.Longitude * math.Pi / 180
	lat2 := loc2.Latitude * math.Pi / 180
	lon2 := loc2.Longitude * math.Pi / 180

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return bm.earthRadius * c
}

// CheckGeofence verifies if a router is allowed to connect from its current physical location.
// Returns an error if the device drifted beyond its allowed radius (geofence violation).
func (bm *BillingManager) CheckGeofence(deviceID string, currentLocation Location) error {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	// 1. Check for active emergency overrides first
	if override, exists := bm.overrides[deviceID]; exists {
		if time.Now().Before(override.ExpiresAt) {
			// Override is valid, bypass standard geofencing
			return nil
		}
	}

	sub, exists := bm.subscriptions[deviceID]
	if !exists {
		// Exception: If no subscription is found, deny access to protect resources
		return errors.New("no active subscription found for device")
	}

	// 2. Mobile plans are inherently unrestricted geographically
	if sub.Plan == PlanMobile {
		return nil
	}

	// 3. Fixed plans require a strictly defined Home Location
	if sub.Plan == PlanFixed {
		if sub.HomeLocation == nil {
			return errors.New("system error: fixed plan missing home location")
		}

		driftKm := bm.calculateDistance(*sub.HomeLocation, currentLocation)
		if driftKm > sub.AllowedRadiusKm {
			return fmt.Errorf("geofence violation: device moved %.2f km from home (limit: %.2f km)", driftKm, sub.AllowedRadiusKm)
		}
	}

	return nil
}

// --- REST API HANDLERS ---

// OverrideGeofenceHandler processes POST /api/geofence/override (Admin Only)
func (bm *BillingManager) OverrideGeofenceHandler(c *gin.Context) {
	var req GeofenceOverrideReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload", "details": err.Error()})
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	overrideID := uuid.New().String() // Generate a unique ID for the override event
	expiresAt := time.Now().Add(time.Duration(req.TTLS) * time.Second)

	bm.overrides[req.DeviceID] = &GeofenceOverride{
		OverrideID: overrideID,
		DeviceID:   req.DeviceID,
		ExpiresAt:  expiresAt,
		Reason:     req.Reason,
	}

	// Audit trail for admin action
	fmt.Printf("[ADMIN ACTION] Geofence overridden for Device:%s TTL:%ds Reason:%s\n", req.DeviceID, req.TTLS, req.Reason)

	c.JSON(http.StatusOK, gin.H{
		"override_id": overrideID,
		"expires_at":  expiresAt.Format(time.RFC3339),
	})
}

// IngestBillingEventHandler processes POST /api/billing/event (Idempotent tracking)
func (bm *BillingManager) IngestBillingEventHandler(c *gin.Context) {
	var req BillingEventReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format"})
		return
	}

	// Exception Handling: Validate event type
	if req.Type != "start" && req.Type != "stop" && req.Type != "usage" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event type. Must be start, stop, or usage"})
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Idempotency check: Reject duplicate events to prevent double billing
	if _, exists := bm.billingEvents[req.EventID]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Billing event ID already processed"})
		return
	}

	// Store the event for ISP accounting database sync
	bm.billingEvents[req.EventID] = &BillingEventRecord{
		EventID:    req.EventID,
		EventType:  req.Type,
		SessionID:  req.SessionID,
		DeviceID:   req.DeviceID,
		BytesDelta: req.BytesDelta,
		Timestamp:  req.Timestamp,
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id": req.EventID,
		"status":   "accepted",
	})
}

// GetGeofenceStatusHandler processes GET /api/geofence/status
func (bm *BillingManager) GetGeofenceStatusHandler(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	bm.mu.RLock()
	defer bm.mu.RUnlock()

	sub, exists := bm.subscriptions[deviceID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "No subscription found"})
		return
	}

	// Determine status based on plan and overrides
	status := "restricted"
	if sub.Plan == PlanMobile {
		status = "unrestricted_mobile"
	} else {
		// Check for active override
		if override, hasOverride := bm.overrides[deviceID]; hasOverride && time.Now().Before(override.ExpiresAt) {
			status = "override_active"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"plan":      sub.Plan,
		"status":    status,
	})
}

// AddMockSubscription is a helper function to inject test data without DB connection
func (bm *BillingManager) AddMockSubscription(sub *SubscriptionRecord) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.subscriptions[sub.DeviceID] = sub
}
