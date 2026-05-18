package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("--- Starting VNU-LEO Core Network ---")

	// 1. Load Deployment Settings (The Entry Point)
	deployPath := "data/deploy_setting.json"
	deployRaw, err := ioutil.ReadFile(deployPath)
	if err != nil {
		log.Fatalf("Critical: Could not read deploy config at %s: %v\n", deployPath, err)
	}

	var deployConfig DeployConfig
	if err := json.Unmarshal(deployRaw, &deployConfig); err != nil {
		log.Fatalf("Critical: Failed to parse deploy JSON: %v\n", err)
	}

	// 2. Load System Settings (Physics & Logic params)
	log.Printf("Loading system settings from: %s\n", deployConfig.SystemSettingsPath)
	systemRaw, err := ioutil.ReadFile(deployConfig.SystemSettingsPath)
	if err != nil {
		log.Fatalf("Critical: Could not read system settings: %v\n", err)
	}

	var systemSettings SystemSettings
	if err := json.Unmarshal(systemRaw, &systemSettings); err != nil {
		log.Fatalf("Critical: Failed to parse system settings JSON: %v\n", err)
	}

	// Khởi tạo Module chống giả mạo thiết bị
	provisioningManager := NewProvisioningManager(systemSettings)

	// 3. Initialize Gateway Pool with System Settings
	gatewayPool := NewGatewayPool(systemSettings)

	// 4. Load Gateway Data from the path specified in deployConfig
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

	// 5. Initialize Handover Manager
	handoverManager := NewHandoverManager(gatewayPool, systemSettings)
	log.Println("Handover Manager is online.")

	// 6. REST API Setup
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Add CORS middleware
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

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "ok",
				"satellite": systemSettings.DefaultSatelliteID,
				"port":      deployConfig.AppPort,
			})
		})

		v1.GET("/gateways", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": gatewayPool.GetAllGateways()})
		})

		v1.GET("/sessions", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": handoverManager.GetActiveSessions()})
		})

		v1.POST("/router/connect", func(c *gin.Context) {
			var req struct {
				MAC string `json:"router_mac" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
				return
			}
			sess, err := handoverManager.HandleRouterConnect(req.MAC)
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"session": sess})
		})

		v1.POST("/ephemeris/update", func(c *gin.Context) {
			var update EphemerisUpdate
			if err := c.ShouldBindJSON(&update); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ephemeris data"})
				return
			}
			log.Printf("Received ephemeris update: %d satellite records\n", len(update.Data))
			c.JSON(http.StatusOK, gin.H{"status": "ephemeris updated", "count": len(update.Data)})
		})

		// ROUTE DEVICE PROVISIONING (Chuyển vào trong block v1 cho đúng cấu trúc)
		v1.POST("/devices/register", provisioningManager.RegisterHandler)
		v1.POST("/devices/verify", provisioningManager.VerifyHandler)
		v1.POST("/devices/revoke", provisioningManager.RevokeHandler)
	}

	// 7. Start the Server using the port from deployConfig
	fullPort := ":" + deployConfig.AppPort
	log.Printf("VNU-LEO Core API listening on %s\n", fullPort)
	if err := router.Run(fullPort); err != nil {
		log.Fatalf("Critical Error: Failed to start server: %v\n", err)
	}
}
