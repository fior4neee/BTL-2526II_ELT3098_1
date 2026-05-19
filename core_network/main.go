package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

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
	if deployConfig.DeviceDataPath == "" {
		deployConfig.DeviceDataPath = "data/devices.json"
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

	provisioningManager := NewProvisioningManager(systemSettings)
	if err := provisioningManager.LoadDevicesFromFile(deployConfig.DeviceDataPath); err != nil {
		log.Printf("Warning: could not load device registry from %s: %v\n", deployConfig.DeviceDataPath, err)
	} else {
		log.Printf("Successfully loaded %d device records.\n", len(provisioningManager.ListDevices()))
	}

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

	handoverManager := NewHandoverManager(gatewayPool, systemSettings)
	log.Println("Handover Manager is online.")
	ephemerisStore := NewEphemerisStore()

	// Background reconciler: if no ephemeris updates are received for a threshold,
	// clear all sessions to ensure gateway session counters return to 0 when
	// orbit_calc/ephemeris publisher and simulators are stopped.
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		staleThreshold := 30 * time.Second
		for range ticker.C {
			last := ephemerisStore.LastUpdate()
			if last.IsZero() || time.Since(last) > staleThreshold {
				cleared := handoverManager.ClearAllSessions()
				if cleared > 0 {
					log.Printf("Reconciler: cleared %d sessions due to stale ephemeris (last update: %v)", cleared, last)
				}
			}
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	registerAPIRoutes(router, gatewayPool, handoverManager, provisioningManager, ephemerisStore, systemSettings, deployConfig)

	fullPort := ":" + deployConfig.AppPort
	log.Printf("VNU-LEO Core API listening on %s\n", fullPort)
	if err := router.Run(fullPort); err != nil {
		log.Fatalf("Critical Error: Failed to start server: %v\n", err)
	}
}
