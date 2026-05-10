# API Reference (Phase 1 & Phase 2)

All endpoints use HTTPS.

## Phase 1 (Core) APIs

| Method | HTTPS Path | Description + Responses (200 and 400 variants) |
|---|---|---|
| GET | /api/gateways | <ul style="list-style-type: none; padding-left: 0;"><li>Returns all gateway stations with live status, load, and signal metrics.</li><li>`200`: list of gateways with health, capacity, and current satellite link stats.</li><li>`400 Bad Request`: invalid query filters.</li><li>`404 Not Found`: gateway set unavailable.</li></ul> |
| GET | /api/satellites | <ul style="list-style-type: none; padding-left: 0;"><li>Returns current satellite states used for elevation and link quality.</li><li>`200`: list of satellites with ephemeris-derived position and elevation per gateway.</li><li>`400 Bad Request`: invalid time window.</li><li>`404 Not Found`: no satellite data loaded.</li></ul> |
| GET | /api/sessions | <ul style="list-style-type: none; padding-left: 0;"><li>Returns active user sessions across gateways.</li><li>`200`: array of sessions with gateway, satellite, C/N, duration, and state.</li><li>`400 Bad Request`: invalid pagination.</li><li>`404 Not Found`: no sessions match.</li></ul> |
| GET | /api/handovers | <ul style="list-style-type: none; padding-left: 0;"><li>Returns recent handover events.</li><li>`200`: list of handover events with timings, old/new gateway, duration, packet loss.</li><li>`400 Bad Request`: invalid range.</li><li>`404 Not Found`: no events in range.</li></ul> |
| POST | /api/handover/trigger | <ul style="list-style-type: none; padding-left: 0;"><li>Manually initiates a handover for a session (for demo/testing).</li><li>`200`: handover accepted with planned target gateway.</li><li>`400 Bad Request`: session not eligible.</li><li>`409 Conflict`: handover already in progress.</li></ul> |
| GET | /api/telemetry/stream | <ul style="list-style-type: none; padding-left: 0;"><li>WebSocket endpoint for real-time gateway/session/handover telemetry.</li><li>`200`: websocket upgrade accepted and stream begins.</li><li>`400 Bad Request`: invalid token or missing parameters.</li><li>`401 Unauthorized`: authentication failed.</li></ul> |
| GET | /api/health | <ul style="list-style-type: none; padding-left: 0;"><li>Returns service health and build metadata.</li><li>`200`: status OK with version and uptime.</li><li>`400 Bad Request`: unsupported probe type.</li></ul> |

## Phase 2 (Advanced/Optional) APIs

| Method | HTTPS Path | Description + Responses (200 and 400 variants) |
|---|---|---|
| POST | /api/devices/register | <ul style="list-style-type: none; padding-left: 0;"><li>Registers a router device (MAC/Hardware ID) and issues a provisioning token.</li><li>`200`: device created with token and status.</li><li>`400 Bad Request`: invalid identifiers.</li><li>`409 Conflict`: device already registered.</li></ul> |
| POST | /api/devices/verify | <ul style="list-style-type: none; padding-left: 0;"><li>Verifies device certificate on first connection.</li><li>`200`: verification successful and device activated.</li><li>`400 Bad Request`: invalid certificate.</li><li>`401 Unauthorized`: signature mismatch.</li></ul> |
| POST | /api/devices/revoke | <ul style="list-style-type: none; padding-left: 0;"><li>Revokes a device and blocks further access.</li><li>`200`: device revoked with timestamp.</li><li>`400 Bad Request`: invalid device ID.</li><li>`404 Not Found`: device not found.</li></ul> |
| GET | /api/billing/usage | <ul style="list-style-type: none; padding-left: 0;"><li>Returns usage records by device or account.</li><li>`200`: usage list with timestamps and totals.</li><li>`400 Bad Request`: invalid filter or date range.</li><li>`404 Not Found`: no records.</li></ul> |
| POST | /api/billing/event | <ul style="list-style-type: none; padding-left: 0;"><li>Ingests a billing event (session start/stop, usage delta).</li><li>`200`: event accepted and stored.</li><li>`400 Bad Request`: malformed event.</li><li>`409 Conflict`: duplicate event ID.</li></ul> |
| GET | /api/geofence/status | <ul style="list-style-type: none; padding-left: 0;"><li>Returns geo-fence status for a device or session.</li><li>`200`: inside/outside status with last location.</li><li>`400 Bad Request`: missing device/session.</li><li>`404 Not Found`: no geo-fence rule.</li></ul> |
| POST | /api/geofence/override | <ul style="list-style-type: none; padding-left: 0;"><li>Temporarily overrides geo-fence enforcement (admin use).</li><li>`200`: override applied with TTL.</li><li>`400 Bad Request`: invalid TTL or scope.</li><li>`403 Forbidden`: insufficient role.</li></ul> |

## Note

APIs live in `core_network`.

`orbit_calc` exports data files (TLE/JSON).

For real-time satellite and link data, still need a telemetry/WebSocket stream from `core_network` to `client_app` for antenna pointing and signal metrics.