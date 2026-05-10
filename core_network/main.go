package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// loadGatewaysFromFile reads and parses the JSON gateway data
func loadGatewaysFromFile(path string) ([]Gateway, error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var gateways []Gateway
	err = json.Unmarshal(byteValue, &gateways)
	return gateways, err
}

func main() {
	log.Println("--- VNU-LEO Core Network Booting ---")

	// 1. Initialize Gateway Pool
	gatewayPool := NewGatewayPool()

	// 2. Load data from folder /data/gateways.json
	gateways, err := loadGatewaysFromFile("data/gateways.json")
	if err != nil {
		log.Fatalf("Critical: Could not load gateway data: %v\n", err)
	}

	for _, g := range gateways {
		gwCopy := g // Avoid pointer issues in range
		gatewayPool.AddGateway(&gwCopy)
	}
	log.Printf("Successfully loaded %d gateways from local storage.\n", len(gateways))

	// 3. Initialize Handover Manager
	handoverManager := NewHandoverManager(gatewayPool)
	log.Println("Handover Manager is online.")

	// 4. REST API Setup
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		v1.GET("/gateways", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": gatewayPool.GetAllGateways()})
		})

		v1.GET("/sessions", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": handoverManager.GetActiveSessions()})
		})

		v1.POST("/router/connect", func(c *gin.Context) {
			var req struct {
				MAC string  `json:"router_mac" binding:"required"`
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
				return
			}
			loc := Location{Latitude: req.Lat, Longitude: req.Lon}
			sess, err := handoverManager.HandleRouterConnect(req.MAC, loc)
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"session": sess})
		})
	}

	port := ":8080"
	log.Printf("VNU-LEO Core API listening on %s\n", port)
	router.Run(port)
}
