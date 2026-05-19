# web_admin - VNU-LEO ISP Admin Dashboard

**Technology**: SvelteKit 2.x + TailwindCSS 3.x + DaisyUI 4.x  
**Scope**: API-backed gateway, session, satellite, handover, and device monitoring

## Overview

The web admin is an operator dashboard for the VNU-LEO Core Network. It reads live data from the Go backend instead of maintaining random browser-side mocks.

The UI intentionally does not include billing, alert feeds, time-series charts, admin audit logs, or static RBAC screens because those modules are not implemented by the current backend.

## Project Structure

```text
web_admin/
├── src/
│   ├── app.css
│   ├── app.html
│   ├── lib/
│   │   ├── types.ts
│   │   ├── api/
│   │   │   └── client.ts
│   │   ├── stores/
│   │   │   └── index.ts
│   │   └── components/
│   │       ├── Sidebar.svelte
│   │       ├── Topbar.svelte
│   │       └── ConstellationMap.svelte
│   └── routes/
│       ├── +layout.svelte
│       ├── +page.svelte
│       ├── monitoring/
│       │   └── +page.svelte
│       └── security/
│           └── +page.svelte
├── package.json
├── svelte.config.js
├── vite.config.ts
└── tailwind.config.js
```

## Quick Start

```bash
cd core_network
go run .
```

```bash
cd web_admin
npm install
npm run dev
```

The default Core API base is `http://localhost:8080/api/v1`. Override it with:

```bash
VITE_CORE_API_BASE=http://localhost:8080/api/v1 npm run dev
```

## Pages

### `/` - Overview

- Gateway list from `GET /api/gateways`
- Satellite list and map from `GET /api/satellites`
- Empty satellite state until ephemeris is posted to the backend

### `/monitoring` - Monitoring

- Gateway status cards from `GET /api/gateways`
- Session table from `GET /api/sessions`
- Handover table from `GET /api/handovers`
- Manual handover action through `POST /api/handover/trigger`

### `/security` - Devices

- Device registry from `GET /api/devices`
- Device suspension through `POST /api/devices/{mac}/suspend`
- Device revocation through `POST /api/devices/revoke`

## Backend Endpoints Used

| Method | Endpoint | Used for |
|---|---|---|
| `GET` | `/api/gateways` | Gateway list and map pins |
| `GET` | `/api/sessions` | Session monitoring |
| `GET` | `/api/satellites` | Satellite list and map positions |
| `GET` | `/api/handovers` | Handover history |
| `POST` | `/api/handover/trigger` | Manual handover demo action |
| `GET` | `/api/telemetry/stream` | Server-sent telemetry refresh hint |
| `GET` | `/api/devices` | Device registry |
| `POST` | `/api/devices/{mac}/suspend` | Device suspension |
| `POST` | `/api/devices/revoke` | Device revocation |

## Validation

```bash
npm run check
npm run build
```

The backend should also pass:

```bash
go test ./...
```
