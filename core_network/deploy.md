Dưới đây là bản dịch toàn bộ nội dung file `DEPLOY.md` sang tiếng Anh theo đúng cấu trúc và các bản cập nhật mới nhất của bạn:

```markdown
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

### Step 2: Install Gin Web Framework

The system uses the **Gin** framework to build the REST API. Download it by running:

```bash
go get -u [github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)

```

*Note: This command will automatically generate a `go.sum` file and update your `go.mod` file.*

### Step 3: Prepare Directory Structure and Data

---

## 🚀 Running the System

Ensure you are still inside the `core_network` directory, then execute the following command:

```bash
go run .

```

If successful, the system will output the following logs:

```text
--- VNU-LEO Core Network Booting ---
Successfully loaded 1 gateways from local storage.
Handover Manager is online.
VNU-LEO Core API listening on :8080

```

---

## 🧪 Testing

Once the server is up and running (listening on port 8080), you can use a web browser or Postman to test the endpoints:

1. **Health Check:**
* **URL:** `http://localhost:8080/api/v1/health`
* **Expected Result:** `{"status":"ok"}`


2. **Get Gateways List:**
* **URL:** `http://localhost:8080/api/v1/gateways`
* **Purpose:** Verify that the JSON data was loaded successfully.


3. **Simulate Router Connection (POST Request):**
* Use Postman to send a POST request to `http://localhost:8080/api/v1/router/connect`
* **Body (JSON):**
```json
{
  "router_mac": "AA:BB:CC:DD:EE:FF",
  "lat": 21.0,
  "lon": 105.8
}

```





---

## 🐞 Troubleshooting

* **Error `StateHold is not a type`:** Ensure you have updated the `models.go` file correctly (change the type to `SessionState`).
* **Error `cannot find module...`:** Run the command `go mod tidy` so Go can automatically download and update any missing dependencies.
* **Error `gateways.json: no such file or directory`:** Double-check that you have created the `data` folder and the `gateways.json` file in the correct location (inside the `core_network` folder).

---

*VNU-LEO Project - 2026*
