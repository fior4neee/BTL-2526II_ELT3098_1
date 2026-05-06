# End-User Router Client - Remediation List

**Module Owner**: Client/Desktop App Team  
**Phase**: P1 (Core) + P2 (Optional)  
**Technology Stack**: Tauri v2 (Rust backend) + Svelte (Frontend)  
**Requirement**: Bài toán Client (End-user Router)

## Current State
- `starsim_notebook.ipynb` is a Starlink orbit visualization (not an end-user router app).
- No antenna tracking, real-time signal visualization, or user-facing UI.
- No C/N, attenuation, or quality metrics displayed.

## Phase 1 Deliverables (Core)

### Tauri Backend (src-tauri/)

#### 1. **attenna_tracker.rs** (note: should be antenna_tracker.rs)
- [ ] Implement `AntennaTracker` struct:
  - Phased-array model (element count, beam steering algorithms)
  - Azimuth/elevation pointing given satellite ephemeris
  - Beam width, gain, sidelobe pattern
- [ ] Calculate beam pointing:
  - Subscribe to satellite position updates from core_network module
  - Compute target azimuth/elevation for user's lat/lon
  - Output steering commands to hypothetical antenna hardware
- [ ] Simulate antenna performance:
  - Calculate achieved gain based on beam pointing accuracy
  - Model beam pattern effects (null steering for interference)
  - Output pointing metrics for UI display (azimuth, elevation, beam quality)
- [ ] Real-time updates: emit pointing data at 10 Hz to Svelte frontend

#### 2. **Signal Quality Module** (in attenna_tracker.rs or separate)
- [ ] Receive link budget parameters from core_network:
  - Satellite EIRP (effective isotropic radiated power)
  - Path loss, atmospheric loss
  - Receiver G/T (gain-to-noise temperature)
- [ ] Calculate in real-time:
  - Received signal strength (dBm)
  - C/N ratio (Carrier-to-Noise)
  - Eb/N0 (energy per bit over noise spectral density)
  - BER or BLER (bit/block error rate) based on modulation scheme
- [ ] Emit metrics to Svelte: {carrier_power_dbm, c_n_ratio_db, modulation_scheme, link_status}

### Svelte Frontend (src/)

#### 3. **SignalDashboard.svelte**
- [ ] Real-time visualization of:
  - **Antenna pointing**: compass (N/S/E/W), elevation angle dial, beam quality bar
  - **Signal metrics**: C/N gauge, received power histogram, BER chart
  - **Satellite info**: current satellite name, elevation angle, time to horizon
  - **Connection status**: connected/searching/outage indicator
- [ ] Historical graph: C/N ratio over last 1 hour (rolling window)
- [ ] Handover indicator: show when handover occurs, which gateway (Hanoi/Danang/HCMC)
- [ ] Layout:
  - Left panel: antenna pointing (compass + elevation dial)
  - Center panel: signal quality gauges (C/N, power, BER)
  - Right panel: satellite timeline (next pass predictions, current satellite info)

#### 4. **Connection Manager UI** (in SignalDashboard.svelte)
- [ ] Manual satellite selection (if multiple above horizon)
- [ ] Connection history: log of connected satellites, duration, data transferred
- [ ] Diagnostics: ping latency, jitter, packet loss readout
- [ ] Settings: signal threshold, beam steering mode (auto/manual), log level

## Phase 2 Enhancements (Optional)

### Tauri Backend (src-tauri/)

#### 5. **geo_reporter.rs**
- [ ] GPS/GNSS receiver integration (simulated or real):
  - Accept location fixes from device
  - Track current latitude/longitude/altitude
  - Update interval: 1 Hz
- [ ] Geo-fencing enforcement:
  - Compare current location to subscription geo-fence bounds
  - If out-of-bounds and subscription is "Fixed", trigger warning or auto-disconnect
  - Report location status to web admin (see web_admin/security)
- [ ] Location history: buffer last 24 hours of locations for audit trail
- [ ] Export: provide location data to billing module for usage attribution

### Svelte Frontend (src/)

#### 6. **DataUsage.svelte**
- [ ] Real-time usage display:
  - Current data rate (Mbps down/up)
  - Session duration
  - Data transferred this session
  - Cumulative monthly usage (if available from backend)
- [ ] Usage chart: 24-hour rolling histogram of data rate
- [ ] Subscription info:
  - Plan name (Fixed/Mobile)
  - Geo-fence status (if Fixed: show fence on map, current location indicator)
  - Data cap and remaining quota
  - Cost estimate (if pay-as-you-go)
- [ ] Alerts:
  - Approaching data limit warning
  - Geo-fence boundary breach (for Fixed plans)
  - Session disconnection reason (if any)

#### 7. **Location Map** (optional in DataUsage.svelte)
- [ ] Show user's current location and subscription geo-fence (if available)
- [ ] Satellite footprint overlay (if internet connectivity available)
- [ ] Map library: Leaflet or Mapbox

## Testing & Validation (P1)

- [ ] Unit tests: antenna pointing math, link budget calculations
- [ ] Integration test: simulate satellite pass, verify antenna tracking and C/N updates in UI
- [ ] UI test: verify all gauges update, historical graphs render correctly
- [ ] Performance: measure CPU/memory under sustained 10 Hz update rate (target: <10% CPU on modern laptop)
- [ ] Demo: show a 15-minute satellite pass with live antenna tracking and signal quality visualization

## Tauri Project Setup (First Time)

- [ ] Initialize Tauri v2 project: `cargo tauri init`
- [ ] Configure allowed paths for logs, cache
- [ ] Build: `cargo tauri build` (desktop binary)
- [ ] Security: review Tauri sandbox and permission model

## Svelte Project Setup (First Time)

 [x] Build & dev: `npm run dev` (dev server), `npm run build` (production)
- Tauri v2 guide: https://v2.tauri.app/
- Phased-array antenna patterns & beam steering (IEEE Trans. Antennas Propag.)
- Svelte documentation: https://svelte.dev/
- Chart.js for real-time visualizations
