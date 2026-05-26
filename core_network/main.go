package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Configure standard upgrade parameters for safe memory handling
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin connection rules for VNU-LEO frontend interfaces
	},
}

func main() {
	log.Println("--- Starting VNU-LEO Core Network ---")

	deployPath := "data/deploy_setting.json"
	deployRaw, err := ioutil.ReadFile(deployPath)
	if err != nil {
		log.Fatalf("Critical: Could not read deploy config at %s: %v\n", deployPath, err)
	}

	var deployConfig DeployConfig
	if err := json.Unmarshal(deployRaw, &deployConfig); err != nil {
		log.Fatalf("Critical: Failed to parse deploy JSON: %v\n", err)
	}

	log.Printf("Loading system settings from: %s\n", deployConfig.SystemSettingsPath)
	systemRaw, err := ioutil.ReadFile(deployConfig.SystemSettingsPath)
	if err != nil {
		log.Fatalf("Critical: Could not read system settings: %v\n", err)
	}

	var systemSettings SystemSettings
	if err := json.Unmarshal(systemRaw, &systemSettings); err != nil {
		log.Fatalf("Critical: Failed to parse system settings JSON: %v\n", err)
	}

	// Load TLE data for satellite propagation
	tlePath := deployConfig.TlePath
	if tlePath == "" {
		timestamped := "outputs/internet/vnu_leo.tle"
		timestampedAlt := "../outputs/internet/vnu_leo.tle"
		if _, err := ioutil.ReadFile(timestamped); err == nil {
			timestampedAlt = timestamped
		}
		tlePath = timestampedAlt
	}
	log.Printf("Loading TLE data from: %s\n", tlePath)
	tleRecords, err := LoadTLE(tlePath)
	if err != nil {
		log.Fatalf("Critical: Could not load TLE file: %v\n", err)
	}
	satTracker := NewSatelliteTracker(tleRecords)
	satTracker.Update(time.Now().UTC())

	provisioningManager := NewProvisioningManager(systemSettings)
	provisioningManager.LoadDevicesFromFile("data/devices.json")

	billingManager := NewBillingManager(systemSettings)

	// Router 001 (Fixed - Hanoi)
	billingManager.AddMockSubscription(&SubscriptionRecord{
		DeviceID:        "router-vnu-leo-001",
		Plan:            PlanFixed,
		AllowedRadiusKm: 50.0,
		HomeLocation:    &Location{Latitude: 21.0285, Longitude: 105.8542, Altitude: 0},
	})

	// Router 002 (Fixed - Hanoi)
	billingManager.AddMockSubscription(&SubscriptionRecord{
		DeviceID:        "router-vnu-leo-002",
		Plan:            PlanFixed,
		AllowedRadiusKm: 50.0,
		HomeLocation:    &Location{Latitude: 21.0285, Longitude: 105.8542, Altitude: 0},
	})

	// Router 003 (Fixed - Da Nang) - Note: Provisioning status is revoked
	billingManager.AddMockSubscription(&SubscriptionRecord{
		DeviceID:        "router-vnu-leo-003",
		Plan:            PlanFixed,
		AllowedRadiusKm: 50.0,
		HomeLocation:    &Location{Latitude: 16.0471, Longitude: 108.2062, Altitude: 0},
	})

	// Router 004 (Mobile - Haiphong Base) - This one will NEVER trigger a geofence error!
	billingManager.AddMockSubscription(&SubscriptionRecord{
		DeviceID:        "router-vnu-leo-004",
		Plan:            PlanMobile,
		AllowedRadiusKm: 0.0,
		HomeLocation:    &Location{Latitude: 20.8449, Longitude: 106.6881, Altitude: 0},
	})

	// Router 005 (Fixed - Can Tho)
	billingManager.AddMockSubscription(&SubscriptionRecord{
		DeviceID:        "router-vnu-leo-005",
		Plan:            PlanFixed,
		AllowedRadiusKm: 50.0,
		HomeLocation:    &Location{Latitude: 10.0452, Longitude: 105.7469, Altitude: 0},
	})

	// Router 006 (Fixed - Nha Trang) - Note: Provisioning status is revoked
	billingManager.AddMockSubscription(&SubscriptionRecord{
		DeviceID:        "router-vnu-leo-006",
		Plan:            PlanFixed,
		AllowedRadiusKm: 50.0,
		HomeLocation:    &Location{Latitude: 12.2388, Longitude: 109.1967, Altitude: 0},
	})

	gatewayPool := NewGatewayPool(systemSettings)

	log.Printf("Loading gateway data from: %s\n", deployConfig.GatewayDataPath)
	gatewayDataRaw, err := ioutil.ReadFile(deployConfig.GatewayDataPath)
	if err != nil {
		log.Fatalf("Critical: Could not read gateway data: %v\n", err)
	}

	var gateways []Gateway
	if err := json.Unmarshal(gatewayDataRaw, &gateways); err != nil {
		log.Fatalf("Critical: Failed to parse gateway JSON: %v\n", err)
	}

	for _, g := range gateways {
		gCopy := g
		gatewayPool.AddGateway(&gCopy)
	}
	log.Printf("Successfully initialized %d gateways.\n", len(gateways))

	handoverManager := NewHandoverManager(gatewayPool, systemSettings, billingManager, provisioningManager, satTracker)
	log.Println("Handover Manager is online with Geofence protection.")

	telemetryEngine := NewTelemetryEngine(handoverManager, gatewayPool, satTracker, 5)
	telemetryEngine.Start()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "port": deployConfig.AppPort})
		})
		v1.GET("/gateways", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"gateways": gatewayPool.GetAllGateways()})
		})
		v1.GET("/sessions", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"sessions": handoverManager.GetActiveSessions()})
		})

		// 1. GET /api/v1/satellites (Matching api.md schemas)
		v1.GET("/satellites", func(c *gin.Context) {
			sats := satTracker.GetStates()
			if len(sats) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "no satellite data loaded"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"satellites": sats})
		})

		// 2. GET /api/v1/handovers (Historical transitions lookup)
		v1.GET("/handovers", func(c *gin.Context) {
			fromRaw := c.Query("from")
			toRaw := c.Query("to")
			gwID := c.Query("gateway_id")

			var fromTime, toTime time.Time
			var err error
			if fromRaw != "" {
				fromTime, err = time.Parse(time.RFC3339, fromRaw)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from timestamp"})
					return
				}
			}
			if toRaw != "" {
				toTime, err = time.Parse(time.RFC3339, toRaw)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to timestamp"})
					return
				}
			}

			handovers := handoverManager.GetHandoverHistory()
			filtered := make([]HandoverEvent, 0, len(handovers))
			for _, h := range handovers {
				if !fromTime.IsZero() && h.Timestamp.Before(fromTime) {
					continue
				}
				if !toTime.IsZero() && h.Timestamp.After(toTime) {
					continue
				}
				if gwID != "" && h.SourceGatewayID != gwID && h.TargetGatewayID != gwID {
					continue
				}
				filtered = append(filtered, h)
			}

			c.JSON(http.StatusOK, gin.H{"handovers": filtered})
		})

		// 3. GET /api/v1/telemetry/stream (WebSocket network pipeline protocol)
		v1.GET("/telemetry/stream", func(c *gin.Context) {
			// Security validation exception check
			token := c.Query("token")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing short-lived authentication token parameter"})
				return
			}
			topicsParam := c.Query("topics")
			topicMap := map[string]bool{}
			for _, t := range strings.Split(topicsParam, ",") {
				if t = strings.TrimSpace(t); t != "" {
					topicMap[t] = true
				}
			}
			if len(topicMap) == 0 {
				topicMap = map[string]bool{"gateways": true, "sessions": true, "handovers": true, "satellites": true, "telemetry": true}
			}

			ws, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				log.Printf("Failed to upgrade server response to websocket context: %v\n", err)
				return
			}

			clientChan := make(chan interface{}, 10)
			handoverManager.registerWs <- wsClient{ch: clientChan, topics: topicMap}

			// Keeps reading/writing loops alive until the subscriber gracefully leaves
			go func() {
				defer func() {
					handoverManager.unregisterWs <- clientChan
					ws.Close()
				}()

				for msg := range clientChan {
					ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
					if err := ws.WriteJSON(msg); err != nil {
						break
					}
				}
			}()
		})

		v1.POST("/router/connect", func(c *gin.Context) {
			var req struct {
				DeviceID string   `json:"device_id" binding:"required"`
				MAC      string   `json:"router_mac" binding:"required"`
				Location Location `json:"location" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
				return
			}
			if status, ok := provisioningManager.GetDeviceStatus(req.DeviceID); ok {
				if status == DeviceSuspended || status == DeviceRevoked {
					c.JSON(http.StatusForbidden, gin.H{"error": "device blocked", "device_status": status})
					return
				}
			}
			sess, err := handoverManager.HandleRouterConnect(req.DeviceID, req.MAC, req.Location)
			if err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"session": sess})
		})

		v1.POST("/router/update", func(c *gin.Context) {
			var req struct {
				DeviceID string   `json:"device_id" binding:"required"`
				Location Location `json:"location" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
				return
			}
			if status, ok := provisioningManager.GetDeviceStatus(req.DeviceID); ok {
				if status == DeviceSuspended || status == DeviceRevoked {
					c.JSON(http.StatusForbidden, gin.H{"error": "device blocked", "device_status": status})
					return
				}
			}
			if err := handoverManager.UpdateDeviceLocation(req.DeviceID, req.Location); err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		v1.POST("/handover/trigger", func(c *gin.Context) {
			var req struct {
				SessionID       string   `json:"session_id" binding:"required"`
				TargetGatewayID string   `json:"target_gateway_id" binding:"required"`
				Location        Location `json:"location" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
				return
			}
			err := handoverManager.TriggerHandover(req.SessionID, req.TargetGatewayID, req.Location)
			if err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "handover_successful"})
		})

		v1.GET("/devices", provisioningManager.ListDevicesHandler)
		v1.POST("/devices/register", provisioningManager.RegisterHandler)
		v1.POST("/devices/verify", provisioningManager.VerifyHandler)
		v1.POST("/devices/activate", provisioningManager.ActivateHandler)
		v1.POST("/devices/revoke", provisioningManager.RevokeHandler)
		v1.POST("/devices/suspend", provisioningManager.SuspendHandler)

		v1.POST("/billing/event", billingManager.IngestBillingEventHandler)
		v1.GET("/geofence/status", billingManager.GetGeofenceStatusHandler)
		v1.POST("/geofence/override", billingManager.OverrideGeofenceHandler)
	}

	fullPort := ":" + deployConfig.AppPort
	log.Printf("VNU-LEO Core API listening on %s\n", fullPort)
	if err := router.Run(fullPort); err != nil {
		log.Fatalf("Critical Error: Failed to start server: %v\n", err)
	}
}
