# Web Admin Dashboard - Remediation List

**Module Owner**: Frontend/DevOps Team  
**Phase**: P1 (Core) + P2 (Optional)  
**Technology Stack**: SvelteKit + TailwindCSS  
**Requirement**: Optional ISP Infrastructure Monitoring

## Current State
- No web admin interface exists.
- Core network module (Go) has no dashboard for ISP operator oversight.

## Phase 1 Deliverables (Core)

### 1. **+page.svelte (Home/Overview Page)**
- [ ] High-level ISP operation view:
  - Network status: # gateways online/offline, # active sessions, # active satellites
  - Top metrics: total traffic volume (last hour/day), average handover duration, packet loss %
  - Alert panel: critical issues (gateway down, excessive handovers, security anomalies)
- [ ] Live sat constellation view:
  - Map showing 3 gateways (Hanoi, Danang, HCMC) as pins
  - Animated satellite positions (if real-time ephemeris available)
  - Color-coded coverage zones (green = good signal, yellow = marginal, red = no coverage)
- [ ] Quick stats cards:
  - Active connections: N sessions
  - Data throughput: X Gbps current
  - Gateway uptime: X.X%
  - Handover rate: Y per minute
- [ ] Link to detailed monitoring (routing to /monitoring path)

### 2. **monitoring/+page.svelte (Detailed Monitoring Dashboard)**
- [ ] Gateway status table:
  - Columns: Gateway Name, Location, Status (alive/dead), # active sessions, CPU%, Memory%, BW usage, Last activity
  - Real-time updates (1 Hz or adjustable)
  - Click gateway row to drill-down into session details
- [ ] Session list:
  - Columns: Session ID, User/Device MAC, Connected Gateway, Duration, Data (MB), Signal Quality (C/N), Current Activity
  - Filter by gateway, status (connected/handover/disconnecting)
  - Real-time updates
- [ ] Handover history timeline:
  - Reverse-chronological list of recent handovers (last 100 events)
  - Columns: Timestamp, Session ID, From Gateway, To Gateway, Duration (ms), Packet Loss (%)
  - Search/filter by time range, session, gateway pair
- [ ] Aggregate graphs (auto-refresh every 5 sec):
  - Traffic volume over time (area chart, last 24 hrs)
  - Handover frequency (bar chart, per hour)
  - Average C/N ratio per gateway (line chart)
  - Active session count trend (line chart)

## Phase 2 Enhancements (Optional)

### 3. **security/+page.svelte (Device Verification & Security)**
- [ ] Device registry:
  - Table of registered routers: MAC, Hardware ID, Model, Owner, Registration Date, Status (active/suspended/revoked)
  - Search by MAC or hardware ID
  - Actions: suspend/revoke device (require admin 2FA confirmation)
  - Import/export device list (CSV)
- [ ] Suspicious activity alerts:
  - Flag unregistered devices attempting connection
  - Geo-fence breaches (if location available from clients)
  - Rapid MAC/Hardware ID changes (spoofing attempt indicator)
  - Display alert history with timestamps
-- [ ] Security operations:
  - Subscription tier summary: # Fixed, # Mobile, # trial
  - Usage dashboard: top 10 users by data volume
  - Geo-fence event log (if enabled): all instances of Fixed customers moving out-of-bounds

### 4. **API Integration**
- [ ] Connect to core_network Go backend (REST endpoints):
  - `GET /api/gateways` → list gateway status
  - `GET /api/sessions` → active sessions + historical
  - `GET /api/handovers` → recent handover events
  - `GET /api/devices` → registered devices + verification status
  - `POST /api/devices/{mac}/revoke` → admin revocation action
- [ ] WebSocket for real-time updates (gateway telemetry, session events, alerts)
- [ ] Authentication: JWT token (issued by core_network, validated by SvelteKit)

### 5. **Admin User Management** (secondary, if time permits)
- [ ] Role-based access control (RBAC):
  - Super Admin: full dashboard + security actions
  - Network Operator: read-only monitoring
- [ ] User list: create/edit/disable admin accounts
- [ ] Session audit log: track all admin actions (logins, revocations, configuration changes)

## UI/UX Considerations

- [ ] Responsive design: usable on desktop and tablet (1024px minimum width)
- [ ] Dark mode option (accessibility)
- [ ] Keyboard shortcuts for common actions (accessibility)
- [ ] Data tables: sortable columns, pagination or infinite scroll
- [ ] Charts: interactive (hover for details, click to filter)
- [ ] Refresh rates configurable by operator (1–60 sec intervals)

## Testing & Validation (P1)

- [ ] Unit tests: components render with mock data
- [ ] Integration test: mock API backend, verify data flows to UI correctly
- [ ] Load test: simulate 1000+ sessions, verify UI responsiveness
- [ ] Demo: show live monitoring of simulated constellation + handover events over 1-hour period

## SvelteKit Project Setup (First Time)

- [ ] Create new SvelteKit project: `npm create svelte@latest web_admin`
- [ ] Choose: Skeleton project (minimal setup)
- [ ] Install dependencies: `npm install`
- [ ] Add UI library:
  - `npm install -D tailwindcss postcss autoprefixer daisyui`
  - `npx tailwindcss init -p`
- [ ] Add charting: `npm install chart.js svelte-chartjs`
- [ ] Dev server: `npm run dev` (http://localhost:5173)
- [ ] Build: `npm run build`

## References
- SvelteKit documentation: https://kit.svelte.dev/
- TailwindCSS: https://tailwindcss.com/
- DaisyUI (Tailwind component library): https://daisyui.com/
- Chart.js for real-time dashboards
- WebSocket integration in SvelteKit (hooks.server.ts)
