# VNU-LEO Web Admin

SvelteKit dashboard for monitoring gateways, sessions, and handovers.

## Prerequisites
- Node.js 18+
- Core Network running on http://localhost:8081

## Run the UI
1. Open a terminal in this folder.
2. Install dependencies:
   - `npm install`
3. Start dev server:
   - `npm run dev`
4. Open the URL printed in the terminal (usually http://localhost:5173).

## Simulate traffic (sessions + handovers)
> Use PowerShell in the repo root or any terminal that can reach the API.

### 1) Check gateways
```powershell
Invoke-RestMethod -Method GET -Uri "http://localhost:8081/api/v1/gateways"
```
Copy a gateway id (example: `GW-HAN-01`, `GW-DAN-01`, `GW-HCM-01`).

### 2) Create a session
```powershell
$body = @{
  device_id = "router-vnu-leo-001"
  router_mac = "AA:BB:CC:DD:EE:01"
  location = @{ lat = 21.0285; lon = 105.8542; alt = 0 }
} | ConvertTo-Json -Depth 4

Invoke-RestMethod -Method POST -Uri "http://localhost:8081/api/v1/router/connect" -Body $body -ContentType "application/json"
```
Copy the returned `session.session_id`.

### 3) Trigger a handover
```powershell
$body = @{
  session_id = "<SESSION_ID_FROM_STEP_2>"
  target_gateway_id = "GW-DAN-01"
  location = @{ lat = 16.0471; lon = 108.2062; alt = 0 }
} | ConvertTo-Json -Depth 4

Invoke-RestMethod -Method POST -Uri "http://localhost:8081/api/v1/handover/trigger" -Body $body -ContentType "application/json"
```

### 4) Verify data in UI
- Overview: gateway counts, sessions, handover metrics
- Monitoring: sessions table and handover history

---

## Notes
- Enable device registry calls (Security tab): set `VITE_ENABLE_DEVICES=true` in your web_admin environment, then restart `npm run dev`.
- Enable live telemetry stream: set `VITE_TELEMETRY_TOKEN=<any-non-empty-string>`, then restart `npm run dev`.
