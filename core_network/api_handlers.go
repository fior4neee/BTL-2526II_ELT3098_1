package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func registerAPIRoutes(
	router *gin.Engine,
	gatewayPool *GatewayPool,
	handoverManager *HandoverManager,
	provisioningManager *ProvisioningManager,
	ephemerisStore *EphemerisStore,
	systemSettings SystemSettings,
	deployConfig DeployConfig,
) {
	api := router.Group("/api")
	{
		api.GET("/health", healthHandler(systemSettings, deployConfig))
		api.GET("/gateways", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"gateways": gatewayPool.GetAllGateways()})
		})
		api.GET("/sessions", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"sessions": handoverManager.GetActiveSessions()})
		})
		api.GET("/satellites", satellitesHandler(ephemerisStore))
		api.GET("/handovers", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"handovers": handoverManager.GetHandoverHistory()})
		})
		api.POST("/handover/trigger", handoverTriggerHandler(handoverManager))
		api.GET("/telemetry/stream", telemetryStreamHandler(gatewayPool))
		api.POST("/ephemeris/update", ephemerisUpdateHandler(ephemerisStore, gatewayPool))
		api.GET("/devices", provisioningManager.ListHandler)
		api.POST("/devices/register", provisioningManager.RegisterHandler)
		api.POST("/devices/verify", provisioningManager.VerifyHandler)
		api.POST("/devices/revoke", provisioningManager.RevokeHandler)
		api.POST("/devices/:mac/suspend", provisioningManager.SuspendByMACHandler)
	}

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler(systemSettings, deployConfig))
		v1.GET("/gateways", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": gatewayPool.GetAllGateways()})
		})
		v1.GET("/sessions", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": handoverManager.GetActiveSessions()})
		})
		v1.GET("/satellites", satellitesHandler(ephemerisStore))
		v1.GET("/handovers", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"handovers": handoverManager.GetHandoverHistory()})
		})
		v1.POST("/handover/trigger", handoverTriggerHandler(handoverManager))
		v1.GET("/telemetry/stream", telemetryStreamHandler(gatewayPool))
		v1.POST("/ephemeris/update", ephemerisUpdateHandler(ephemerisStore, gatewayPool))
		v1.GET("/devices", provisioningManager.ListHandler)
		v1.POST("/devices/register", provisioningManager.RegisterHandler)
		v1.POST("/devices/verify", provisioningManager.VerifyHandler)
		v1.POST("/devices/revoke", provisioningManager.RevokeHandler)
		v1.POST("/devices/:mac/suspend", provisioningManager.SuspendByMACHandler)

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

		v1.POST("/router/disconnect", func(c *gin.Context) {
			var req struct {
				SessionID string `json:"session_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
				return
			}
			if err := handoverManager.HandleRouterDisconnect(req.SessionID); err != nil {
				switch {
				case errors.Is(err, ErrSessionNotFound):
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				default:
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				}
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "disconnected", "session_id": req.SessionID})
		})

		// Admin: clear all sessions (testing only)
		v1.POST("/sessions/clear", func(c *gin.Context) {
			cleared := handoverManager.ClearAllSessions()
			c.JSON(http.StatusOK, gin.H{"status": "cleared", "count": cleared})
		})
	}
}

func healthHandler(systemSettings SystemSettings, deployConfig DeployConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"version":   "1.0.0",
			"satellite": systemSettings.DefaultSatelliteID,
			"port":      deployConfig.AppPort,
		})
	}
}

func satellitesHandler(ephemerisStore *EphemerisStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		satellites := ephemerisStore.List()
		if len(satellites) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "no satellite data loaded", "satellites": []Ephemeris{}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"satellites": satellites})
	}
}

func ephemerisUpdateHandler(ephemerisStore *EphemerisStore, gatewayPool *GatewayPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var update EphemerisUpdate
		if err := c.ShouldBindJSON(&update); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ephemeris data"})
			return
		}
		ephemerisStore.Update(update.Data)
		gatewayPool.LogTelemetry("ephemeris_update", "", gin.H{"count": len(update.Data)}, "ephemeris updated")
		c.JSON(http.StatusOK, gin.H{"status": "ephemeris updated", "count": len(update.Data)})
	}
}

func handoverTriggerHandler(handoverManager *HandoverManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TriggerHandoverReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid handover request"})
			return
		}

		event, err := handoverManager.TriggerManualHandover(req.SessionID, req.TargetGatewayID)
		if err != nil {
			switch {
			case errors.Is(err, ErrHandoverInProgress):
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrInvalidHandover):
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"handover_id": event.ID,
			"eta_ms":      event.DurationMs,
			"handover":    event,
		})
	}
}

func telemetryStreamHandler(gatewayPool *GatewayPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		writeEvent := func(name string, payload interface{}) {
			raw, _ := json.Marshal(payload)
			_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", name, raw)
			flusher.Flush()
		}

		writeEvent("connected", gin.H{"timestamp": time.Now()})

		for {
			select {
			case event := <-gatewayPool.TelemetryCh:
				writeEvent("telemetry", event)
			case <-time.After(15 * time.Second):
				writeEvent("heartbeat", gin.H{"timestamp": time.Now()})
			case <-c.Request.Context().Done():
				return
			}
		}
	}
}
