# Web Admin Remediation List

**Module Owner**: Frontend/DevOps Team  
**Technology Stack**: SvelteKit + TailwindCSS  
**Current Scope**: Core Network monitoring and device controls

## Implemented Alignment

- [x] Gateway list now targets `GET /api/gateways`.
- [x] Session list now targets `GET /api/sessions`.
- [x] Satellite list/map now targets `GET /api/satellites`.
- [x] Handover history now targets `GET /api/handovers`.
- [x] Manual handover action now targets `POST /api/handover/trigger`.
- [x] Telemetry stream connection now targets `GET /api/telemetry/stream`.
- [x] Device registry now targets `GET /api/devices`.
- [x] Device suspension now targets `POST /api/devices/{mac}/suspend`.
- [x] Device revocation uses the existing `POST /api/devices/revoke`.

## Removed From Current Scope

- [x] Network summary cards based on browser-only mock stats.
- [x] Time-series charts for traffic, sessions, handovers, and C/N history.
- [x] Alert panel and security alert feed.
- [x] Admin audit log entries.
- [x] Static Admin RBAC page.
- [x] Billing UI, billing API references, and chart dependencies.

## Remaining Backend Integration Notes

- The web admin defaults to `http://localhost:8081/api`.
- Set `VITE_CORE_API_BASE` to point at another Core Network instance.
- `/api` is the documented route prefix; `/api/v1` remains available for current scripts.
- The telemetry endpoint is implemented as a server-sent event stream.

## Validation Checklist

- [ ] `go test ./...`
- [ ] `npm run check`
- [ ] `npm run build`
- [ ] Smoke-test gateway/session/device list endpoints.
- [ ] Smoke-test device suspend by MAC.
- [ ] Smoke-test router connect followed by manual handover trigger.
- [ ] Smoke-test ephemeris update followed by satellite list.
- [ ] Smoke-test telemetry stream connection.
