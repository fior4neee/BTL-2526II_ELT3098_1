
---

# 🛰️ VNU-LEO Core Network - Deployment Guide

This document provides detailed instructions to set up and run the **Core Network** module of the VNU-LEO project on your local machine.

## 📋 Prerequisites

### 1. Install Go Programming Language
- **Download Link:** [https://go.dev/dl/](https://go.dev/dl/)
- After the installation is complete, open your Terminal (or Command Prompt) and type the following command to verify:
  ```bash
  go version

```

---

## 🛠️ Setup Steps

### Step 1: Initialize Go Module

Go Module is the dependency manager for Go. You need to initialize it before installing any libraries.
Open your Terminal, **navigate to the `core_network` directory**, and run:

```bash
cd core_network
go mod init vnu_leo_core

```

### Step 2: Install Required Dependencies

The system relies on several external packages: **Gin** (for the REST API), **Gorilla WebSocket** (for real-time telemetry streaming), and **Google UUID** (for unique identifier generation). Download them and clean up your module file by running:

```bash
go get -u [github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)
go get -u [github.com/gorilla/websocket](https://github.com/gorilla/websocket)
go get -u [github.com/google/uuid](https://github.com/google/uuid)
go mod tidy

```

*Note: These commands will automatically generate a `go.sum` file and update your `go.mod` file.*

### Step 3: Run the Server

Start the Core Network API by executing the following command in the same directory:

```bash
go run .

```

If successful, the system will output the following logs:

```text
--- Starting VNU-LEO Core Network ---
Loading system settings from: data/system_settings.json
Loading gateway data from: data/gateways.json
Successfully initialized 3 gateways.
Handover Manager is online with Geofence protection.
VNU-LEO Core API listening on :8080

```

---

## 🧪 Testing

Once the server is up and running (listening on port 8080), you can use a web browser or Postman to test the endpoints:

1. **Health Check:**

* **URL:** `http://localhost:8080/api/v1/health`
* **Expected Result:** `{"port":"8080","status":"ok"}`

2. **Get Gateways List:**

* **URL:** `http://localhost:8080/api/v1/gateways`
* **Purpose:** Verify that the JSON configuration data was loaded successfully.

3. **Simulate Router Connection (POST Request):**

* Use Postman to send a POST request to `http://localhost:8080/api/v1/router/connect`
* **Body (JSON):**

```json
{
  "device_id": "router-vnu-leo-001",
  "router_mac": "00:1B:44:11:3A:B7",
  "location": {
    "lat": 21.0285,
    "lon": 105.8542,
    "alt": 0
  }
}

```

---

## 🐞 Troubleshooting

* **Error: `The system cannot find the file specified**`
Check your `data/deploy_setting.json` file to ensure `settings_path` is correctly pointing to `data/system_settings.json`.
* **Error: `undefined: uuid` or `could not import...**`
You forgot to run the dependency installation commands in **Step 2**. Run `go mod tidy` to fix missing packages.

```

```