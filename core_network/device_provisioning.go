package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ProvisioningManager handles device identity, security, and auditing
type ProvisioningManager struct {
	mu           sync.RWMutex
	devices      map[string]*DeviceRecord
	secretSalt   string
	auditLogPath string
}

// NewProvisioningManager initializes the hardware root-of-trust manager
func NewProvisioningManager(settings SystemSettings) *ProvisioningManager {
	salt := settings.ISPSecretSalt
	// Exception Handling: Fallback if salt is missing from config
	if salt == "" {
		salt = "DEFAULT-FALLBACK-SALT-DO-NOT-USE-IN-PROD"
		fmt.Println("WARNING: ISPSecretSalt is missing. Using fallback salt!")
	}

	return &ProvisioningManager{
		devices:      make(map[string]*DeviceRecord),
		secretSalt:   salt,
		auditLogPath: settings.AuditLogPath,
	}
}

// logAudit writes connection and security events to the audit log file safely
func (pm *ProvisioningManager) logAudit(deviceID, action, details string) {
	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("[%s] DEVICE:%s ACTION:%s DETAILS:%s\n", timestamp, deviceID, action, details)

	// Print to console for real-time monitoring
	fmt.Print("AUDIT TRAIL -> " + logEntry)

	if pm.auditLogPath != "" {
		// Exception Handling: Ensure directory exists before opening file
		dir := filepath.Dir(pm.auditLogPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("ERROR: Failed to create audit log directory: %v\n", err)
			return
		}

		// Append to audit log file securely
		f, err := os.OpenFile(pm.auditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("ERROR: Failed to write to audit log: %v\n", err)
			return
		}
		defer f.Close()
		f.WriteString(logEntry)
	}
}

// IsValidMAC checks if the provided string is a valid MAC address format
func IsValidMAC(mac string) bool {
	re := regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	return re.MatchString(mac)
}

// RegisterHandler processes POST /api/v1/devices/register
func (pm *ProvisioningManager) RegisterHandler(c *gin.Context) {
	var req RegisterDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Exception Handling: Validate inputs
	if !IsValidMAC(req.MAC) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid MAC address format"})
		return
	}
	if len(req.HWID) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hardware ID is too short"})
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check for existing device to prevent duplication/spoofing attempts
	if _, exists := pm.devices[req.DeviceID]; exists {
		pm.logAudit(req.DeviceID, "REGISTER_FAILED", "Device ID already exists")
		c.JSON(http.StatusConflict, gin.H{"error": "Device already registered"})
		return
	}

	// Generate Provisioning Token using Hash(HWID + SecretSalt)
	hash := sha256.Sum256([]byte(req.HWID + pm.secretSalt))
	token := hex.EncodeToString(hash[:])

	newDevice := &DeviceRecord{
		DeviceID:          req.DeviceID,
		MACAddress:        req.MAC,
		HardwareID:        req.HWID,
		Status:            DeviceRegistered,
		ProvisioningToken: token,
		RegisteredAt:      time.Now(),
	}

	pm.devices[req.DeviceID] = newDevice
	pm.logAudit(req.DeviceID, "REGISTER_SUCCESS", "Device provisioned with token")

	c.JSON(http.StatusOK, gin.H{
		"device_id":          newDevice.DeviceID,
		"status":             newDevice.Status,
		"provisioning_token": token,
	})
}

// VerifyHandler processes POST /api/v1/devices/verify
func (pm *ProvisioningManager) VerifyHandler(c *gin.Context) {
	var req VerifyDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	device, exists := pm.devices[req.DeviceID]
	if !exists {
		pm.logAudit(req.DeviceID, "VERIFY_FAILED", "Unknown device ID")
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Reject if the device has been blacklisted by ISP
	if device.Status == DeviceRevoked {
		pm.logAudit(req.DeviceID, "VERIFY_DENIED", "Blacklisted device attempted access")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Device is revoked/blacklisted"})
		return
	}

	// Verify certificate and signature (Mock logic for X.509 verification)
	if req.CSRPEM == "" || req.Signature == "" {
		pm.logAudit(req.DeviceID, "VERIFY_FAILED", "Missing certificate or signature")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid certificate or signature"})
		return
	}

	device.Status = DeviceActive
	pm.logAudit(req.DeviceID, "VERIFY_SUCCESS", "Certificate verified, device activated")

	c.JSON(http.StatusOK, gin.H{
		"device_id": device.DeviceID,
		"status":    device.Status,
	})
}

// RevokeHandler processes POST /api/v1/devices/revoke
func (pm *ProvisioningManager) RevokeHandler(c *gin.Context) {
	var req RevokeDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	device, exists := pm.devices[req.DeviceID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Update state to Revoked
	now := time.Now()
	device.Status = DeviceRevoked
	device.RevokedAt = &now

	pm.logAudit(req.DeviceID, "REVOKED", req.Reason)

	c.JSON(http.StatusOK, gin.H{
		"device_id": device.DeviceID,
		"status":    device.Status,
		"timestamp": now.Format(time.RFC3339),
	})
}
