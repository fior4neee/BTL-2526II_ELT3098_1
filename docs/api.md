# API Reference

The Core Network exposes the documented API under `/api`. Existing `/api/v1` routes remain available as compatibility aliases for current deploy scripts and the orbit publisher.

Authentication is not enforced yet because the backend has no auth service.

## Core Monitoring

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/health` | Service health, version, selected satellite, and port. |
| `GET` | `/api/gateways` | Gateway stations with location, status, capacity, antenna model, and session load. |
| `GET` | `/api/sessions` | Active router sessions with gateway, satellite, state, and start time. |
| `GET` | `/api/satellites` | Latest ephemeris records received from `POST /api/ephemeris/update`. Returns `404` until data is loaded. |
| `GET` | `/api/handovers` | Recent handover events, newest first. |
| `POST` | `/api/handover/trigger` | Manually triggers a demo handover for an active session. |
| `GET` | `/api/telemetry/stream` | Server-sent event stream for live gateway, session, handover, and ephemeris events. |
| `POST` | `/api/ephemeris/update` | Receives satellite state updates from `orbit_calc`. |

### `GET /api/gateways`

Response:

```json
{
  "gateways": [
    {
      "id": "GW-HAN-01",
      "name": "Hanoi Gateway",
      "location": { "lat": 21.0285, "lon": 105.8542, "alt": 0.015 },
      "max_sessions": 50000,
      "current_sessions": 12,
      "min_elevation_deg": 25,
      "antenna": { "gain_dbi": 42.5, "beam_width_deg": 1.5 },
      "status": "alive"
    }
  ]
}
```

### `GET /api/sessions`

Response:

```json
{
  "sessions": [
    {
      "session_id": "SES-1001",
      "router_mac": "AA:BB:CC:DD:EE:FF",
      "satellite_id": "SAT-VNU-01",
      "current_gateway_id": "GW-HAN-01",
      "state": "Hold",
      "start_time": "2026-05-19T08:30:00Z"
    }
  ]
}
```

### `POST /api/handover/trigger`

Request:

```json
{
  "session_id": "SES-1001",
  "target_gateway_id": "GW-DAN-01"
}
```

Response:

```json
{
  "handover_id": "HO-1770000000000",
  "eta_ms": 75,
  "handover": {
    "id": "HO-1770000000000",
    "timestamp": "2026-05-19T08:31:00Z",
    "session_id": "SES-1001",
    "from_gateway": "GW-HAN-01",
    "to_gateway": "GW-DAN-01",
    "duration_ms": 75,
    "packet_loss": 0.0005,
    "success": true
  }
}
```

Status codes:

- `200`: handover accepted and completed in the demo state machine.
- `400`: missing payload, unknown session, invalid target, or ineligible session.
- `409`: handover already in progress.

### `POST /api/ephemeris/update`

Request:

```json
{
  "data": [
    {
      "satellite_id": "SAT-VNU-01",
      "latitude": 16.1,
      "longitude": 108.2,
      "altitude": 500,
      "elevation": 42.4,
      "azimuth": 136.7,
      "timestamp": "2026-05-19T08:30:00Z"
    }
  ]
}
```

Response:

```json
{ "status": "ephemeris updated", "count": 1 }
```

## Device Provisioning

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/devices` | Lists registered router devices and verification status. |
| `POST` | `/api/devices/register` | Registers a device and returns a provisioning token. |
| `POST` | `/api/devices/verify` | Verifies a device certificate/signature and activates the device. |
| `POST` | `/api/devices/revoke` | Revokes a device by `device_id`. |
| `POST` | `/api/devices/{mac}/suspend` | Suspends a non-revoked device by MAC address. |

### `GET /api/devices`

Response:

```json
{
  "devices": [
    {
      "device_id": "router-vnu-leo-001",
      "mac": "00:1B:44:11:3A:B7",
      "hw_id": "SEC-ENC-998877A",
      "status": "active",
      "provisioning_token": "a1b2...",
      "registered_at": "2026-05-12T15:33:25Z"
    }
  ]
}
```

### `POST /api/devices/{mac}/suspend`

Response:

```json
{
  "device_id": "router-vnu-leo-001",
  "mac": "00:1B:44:11:3A:B7",
  "status": "suspended",
  "timestamp": "2026-05-19T08:30:00Z"
}
```

Status codes:

- `200`: device suspended.
- `400`: invalid MAC format.
- `404`: device not found.
- `409`: revoked devices cannot be suspended.

## Notes

- `orbit_calc` publishes ephemeris with `POST /api/v1/ephemeris/update` or `POST /api/ephemeris/update`.
- The web admin uses `VITE_CORE_API_BASE` when set, otherwise `http://localhost:8081/api`.
- Billing, geo-fence billing, alert feeds, and admin audit-log APIs are out of scope for the current backend.
