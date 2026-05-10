# web_admin — VNU-LEO ISP Admin Dashboard

**Technology**: SvelteKit 2.x + TailwindCSS 3.x + DaisyUI 4.x + Chart.js  
**Phase**: P1 (Core Monitoring) + P2 (Security & Billing)  
**Owner**: Frontend/DevOps Team

---

## Overview

Real-time ISP operator dashboard for the VNU-LEO satellite constellation. Provides live monitoring of gateways, sessions, handover events, device provisioning, and billing.

## Project Structure

```
web_admin/
├── src/
│   ├── app.css                     Global styles (fonts, CSS vars, utilities)
│   ├── app.html                    Base HTML template
│   ├── lib/
│   │   ├── types.ts                Shared TypeScript interfaces
│   │   ├── api/
│   │   │   └── mock.ts             Mock backend (replace with real Go API calls)
│   │   ├── stores/
│   │   │   └── index.ts            Svelte writable stores + live update engine
│   │   └── components/
│   │       ├── Sidebar.svelte      Navigation sidebar
│   │       ├── Topbar.svelte       Page header with clock + alerts
│   │       ├── StatCard.svelte     Metric card component
│   │       ├── ConstellationMap.svelte  SVG Vietnam map with gateways + satellites
│   │       └── AlertPanel.svelte   Alert list with resolve actions
│   └── routes/
│       ├── +layout.svelte          Root layout (sidebar + live update init)
│       ├── +page.svelte            Home/Overview page (P1)
│       ├── monitoring/
│       │   └── +page.svelte        Detailed monitoring: charts, sessions, handovers (P1)
│       └── security/
│           └── +page.svelte        Security, device registry, billing, RBAC (P2)
├── static/
│   └── favicon.svg
├── package.json
├── svelte.config.js
├── vite.config.ts
├── tailwind.config.js
├── postcss.config.js
└── tsconfig.json
```

## Quick Start

```bash
cd web_admin

# Install dependencies
npm install --legacy-peer-deps

# Start dev server (http://localhost:5173)
npm run dev

# Production build
npm run build
npm run preview
```

## Pages

### `/` — Overview (P1)
- Network stat cards (sessions, traffic, handover latency, packet loss)
- Gateway status cards with live CPU/memory/BW bars
- SVG constellation map (Vietnam, animated satellite positions)
- Alert panel (resolve in-place)

### `/monitoring` — Detailed Monitoring (P1)
- Gateway drill-down cards (click to filter sessions)
- 4 live charts: traffic volume, session count, C/N ratio per gateway, handover frequency
- Session list with filtering + pagination (1 Hz updates)
- Handover history table (last 100, searchable)

### `/security` — Security & Billing (P2)
- **Device Registry**: registered devices, suspend/revoke with confirmation dialog, CSV export
- **Billing Events**: subscription breakdown, top users, geo-fence event log, billing export
- **Security Alerts**: all alerts, resolve button, suspicious activity, admin audit log
- **Admin RBAC**: user list, role permissions matrix, disable/enable accounts

## Connecting to Real Backend

All mock data lives in `src/lib/api/mock.ts`. To connect the real Go backend:

1. Replace store update functions in `src/lib/stores/index.ts` with `fetch()` calls:
   ```typescript
   // Example: replace mock.updateGateways() with:
   const res = await fetch('/api/gateways');
   const data = await res.json();
   gateways.set(data);
   ```

2. WebSocket for real-time updates:
   ```typescript
   const ws = new WebSocket('ws://localhost:8080/ws/telemetry');
   ws.onmessage = (e) => {
     const msg = JSON.parse(e.data);
     if (msg.type === 'gateway_update') gateways.set(msg.data);
     if (msg.type === 'handover_event') handoverHistory.update(h => [msg.data, ...h.slice(0,99)]);
   };
   ```

3. JWT auth header:
   ```typescript
   headers: { 'Authorization': `Bearer ${token}` }
   ```

## API Endpoints (from Core Network Go module)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/gateways` | Gateway status list |
| GET | `/api/sessions` | Active + historical sessions |
| GET | `/api/handovers` | Recent handover events |
| GET | `/api/devices` | Registered devices |
| POST | `/api/devices/{mac}/revoke` | Admin revocation |
| POST | `/api/devices/{mac}/suspend` | Admin suspension |
| GET | `/api/billing/report?from=T0&to=T1` | Billing data |
| WS | `/ws/telemetry` | Real-time stream |

## Success Criteria Checklist (P1)

- [x] Dashboard overview with gateway status, session count, traffic, alerts
- [x] Constellation map with animated satellite positions + gateway pins
- [x] Live monitoring page with 4 chart types (area, line, bar)
- [x] Session list with gateway filter + search + pagination
- [x] Handover history (100 events, searchable, color-coded)
- [x] Alert panel with resolve-in-place
- [x] Real-time updates (1 Hz gateway, 5 Hz sessions via stores)
- [x] Loads in <2 sec (Vite + SvelteKit SSR)
- [x] Responsive (1024px+ tablet/desktop)
- [x] Dark mode (default dark theme)

## Success Criteria Checklist (P2)

- [x] Device registry with suspend/revoke + confirmation 2FA flow
- [x] CSV export for devices and billing reports
- [x] Geo-fence event log
- [x] Billing events table
- [x] RBAC user management table + permissions matrix
- [x] Admin audit log
- [x] Suspicious activity flags
