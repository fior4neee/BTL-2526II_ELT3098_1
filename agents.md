# Team Roles & Tasks (Agents)

## Executive Summary

VNU-LEO is a satellite constellation project with three core technical workstreams (P1) and optional advanced features (P2). This document assigns roles, dependencies, and success criteria for each team.

---

## Phase 1 (Core) - Required Deliverables

### **Team 1: Physics & Orbit (Orbit Calc)**
**Owner**: Physics/Math Lead  
**Language**: Python  
**Deliverables**: 
- `coverage_calculator.py` – Constellation design & coverage validation
- `traffic_model.py` – Link budget, revisit time, throughput analysis
- `simulation.ipynb` – Jupyter notebook for visualization & validation

**Key Tasks**:
1. Implement Walker Delta constellation math (N satellites, plane count, inclination, RAAN spacing)
2. Calculate coverage continuity for Vietnam (8°N–24°N) with min elevation ≥15°
3. Separate analysis for two use cases (Internet/VoIP ~500 km vs. Data/Weather ~1000 km)
4. Validate handover zones (where coverage transitions between satellites)
5. Export satellite TLE and coverage maps for core network integration

**Dependencies**:
- None (can start immediately)

**Interfaces**:
- Output: TLE, coverage statistics → to Core Network for ephemeris feeding

**Success Criteria**:
- [ ] Constellation proposed with valid parameters (N, planes, inclination, spacing)
- [ ] Coverage continuity over all of Vietnam ≥99.9%
- [ ] Handover zones identified (locations & timing)
- [ ] Jupyter notebook runs end-to-end, produces coverage plots + statistics

**Estimated Effort**: 2–3 weeks (1 person part-time, or 1 week full-time)

---

### **Team 2: Core Network & Backend (Go)**
**Owner**: Backend/Systems Lead  
**Language**: Go  
**Deliverables**:
- `gateway_nodes.go` – Gateway infrastructure & state tracking
- `handover_manager.go` – Handover state machine & session management

**Key Tasks**:
1. Define Gateway struct (location, capacity, antenna model, signal thresholds)
2. Implement elevation angle calculation from satellite ephemeris
3. Implement handover state machine (Acquire → Hold → Prepare → Execute → Release)
4. Hysteresis & predictive handover to minimize packet loss
5. Session state tracking across handovers (TCP stateful, UDP best-effort)
6. Handover metrics: duration, packet loss, success rate

**Dependencies**:
- **Soft dependency**: Orbit Calc TLE output (can mock initially, integrate later)

**Interfaces**:
- Input: Satellite ephemeris (TLE/SGP4 state vectors)
- Output: Real-time telemetry (elevation angle, C/N ratio, gateway status) → to Client App & Web Admin
- Telemetry: session events (connect/handover/disconnect) → to Web Admin

**Success Criteria**:
- [ ] Gateway pool manages 3 gateways (Hanoi, Danang, HCMC)
- [ ] Elevation calculation correct (±0.1° vs. reference software)
- [ ] Handover completes in <100 ms with <0.1% packet loss
- [ ] 100+ concurrent sessions supported without CPU saturation
- [ ] Comprehensive logging of all handover events

**Estimated Effort**: 3–4 weeks (2 people part-time, or 2 weeks full-time)

---

### **Team 3: Client Desktop App (Tauri + Svelte)**
**Owner**: Frontend/UI Lead  
**Technology**: Tauri v2 (Rust backend) + Svelte (frontend)  
**Deliverables**:
- `src-tauri/attenna_tracker.rs` – Antenna tracking & signal calculation
- `src/SignalDashboard.svelte` – Real-time UI for signal visualization

**Key Tasks**:
1. Subscribe to satellite positions from Core Network
2. Calculate antenna pointing (azimuth/elevation) for user location
3. Simulate phased-array antenna beam gain
4. Receive real-time signal metrics (C/N, BER, EIRP, path loss)
5. Display antenna pointing on compass + elevation dial
6. Plot real-time C/N gauge, signal strength histogram, BER chart
7. Show satellite timeline (current satellite, next satellite, time-to-horizon)
8. Display handover events with gateway names

**Dependencies**:
- **Hard dependency**: Core Network module (telemetry API, WebSocket)
- Soft dependency: Orbit Calc (can mock satellite positions initially)

**Interfaces**:
- Input: Satellite position updates, link budget metrics (from Core Network)
- Output: User interactions (antenna mode selection, settings) → back to Core Network (optional)

**Success Criteria**:
- [ ] Antenna pointing updates at 10 Hz without lag
- [ ] C/N gauge updates in real-time (1 Hz minimum)
- [ ] UI responsive on 1-hour satellite pass simulation
- [ ] CPU/memory footprint <10% on modern laptop
- [ ] All metrics readable at glance (no overwhelming text)

**Estimated Effort**: 2–3 weeks (1–2 people part-time, or 1.5 weeks full-time)

---

## Phase 2 (Optional) - Advanced Features

### **Team 2B: Device Provisioning & Billing (Go Extension)**
**Owner**: Backend/Security Lead  
**Language**: Go  
**Deliverables**:
- `device_provisioning.go` – Device registration, MAC/Hardware ID verification
- `spaciotemporal_billing.go` – Geo-fence enforcement, subscription tiers, billing events

**Key Tasks**:
1. Device certificate management (registration, revocation)
2. Spoofing detection (MAC/Hardware ID changes)
3. Provisioning workflow integration with gateways
4. Geo-fence boundary definition and enforcement
5. Subscription tier logic (Fixed vs. Mobile)
6. Billing event logging (usage attribution, costs)
7. Admin API for device revocation

**Dependencies**:
- **Hard dependency**: Core Network handover manager (P1)

**Interfaces**:
- Input: Device connection attempts, location from client router (if available)
- Output: Billing events → to Web Admin dashboard

**Success Criteria**:
- [ ] Device registration + verification flow works end-to-end
- [ ] Geo-fence breach detected and logged
- [ ] Billing events generated and auditable
- [ ] Device revocation effective within <10 sec

**Estimated Effort**: 2 weeks (1 person)

---

### **Team 3B: Data Usage & Location Tracking (Tauri + Svelte Extension)**
**Owner**: Frontend/UI Lead  
**Language**: Rust (Tauri) + Svelte  
**Deliverables**:
- `src-tauri/geo_reporter.rs` – GPS integration, geo-fence tracking
- `src/DataUsage.svelte` – Data rate visualization, usage alerts

**Key Tasks**:
1. GPS/GNSS receiver integration (simulated or real)
2. Current location reporting to Core Network
3. Geo-fence status display (if subscription is Fixed)
4. Real-time data rate calculation (Mbps down/up)
5. Usage chart (24-hour rolling histogram)
6. Subscription info display (plan name, geo-fence bounds, data cap)
7. Alerts for approaching limit or geo-fence breach

**Dependencies**:
- **Hard dependency**: Core Network provisioning + billing (P2)
- **Soft dependency**: Client App phase 1 (UI framework already in place)

**Interfaces**:
- Input: GPS location, subscription tier, data cap (from Core Network)
- Output: Real-time data usage, location status

**Success Criteria**:
- [ ] Data rate updates every 100 ms (smooth visualization)
- [ ] Geo-fence breach warning pops up <500 ms after event
- [ ] Historical data stored for >24 hours
- [ ] Export usage data to CSV (for user's own records)

**Estimated Effort**: 1.5 weeks (1 person)

---

### **Team 4: ISP Web Admin (SvelteKit)**
**Owner**: Frontend/DevOps Lead  
**Technology**: SvelteKit + TailwindCSS  
**Deliverables** (spans P1 + P2):

**P1 (Core Monitoring)**:
- `src/routes/+page.svelte` – Dashboard overview (gateway status, active sessions, alerts)
- `src/routes/monitoring/+page.svelte` – Detailed monitoring (session list, handover history)

**P2 (Security & Billing)**:
- `src/routes/security/+page.svelte` – Device registry, billing dashboard, geo-fence events

**Key Tasks**:
1. Real-time gateway status board (alive/dead, CPU, memory, BW usage)
2. Active session list with connection details (C/N, duration, data transferred)
3. Handover event timeline (last 100 events with metrics)
4. Aggregate network graphs (traffic, handover frequency, signal quality trends)
5. Device registry for provisioning (add/suspend/revoke devices) [P2]
6. Billing dashboard (subscription summary, usage by customer, cost-per-user) [P2]
7. Geo-fence event log [P2]
8. API integration with Core Network (REST + WebSocket for real-time updates)
9. Role-based access control (Super Admin, Operator, Billing Manager)

**Dependencies**:
- **Hard dependency**: Core Network module (telemetry API, WebSocket, provisioning endpoints)
- **Hard dependency**: Orbit Calc (optional but useful for animated satellite positions)

**Interfaces**:
- Input: Telemetry from Core Network (WebSocket: gateway status, sessions, handover events)
- Input: Billing/security data (REST: devices, geo-fence events, billing reports)
- Output: Admin actions (device revocation, configuration changes) → to Core Network

**Success Criteria**:
- [ ] Dashboard loads in <2 sec, subsequent updates <500 ms
- [ ] All real-time data refreshes ≤5 sec (configurable)
- [ ] Graphs render correctly with 1000+ data points (no lag)
- [ ] Admin can revoke a device and effect takes <10 sec
- [ ] User list & RBAC enforced
- [ ] Billing report exportable as CSV

**Estimated Effort**: 4 weeks (2 people part-time, or 2 weeks full-time) – spans P1+P2

---

## Cross-Team Coordination

### Handoff Points

1. **Orbit Calc → Core Network**: TLE and coverage statistics
   - Format: SGP4-compatible TLE file + JSON coverage metadata
   - Update frequency: once per day (or on-demand for constellation redesigns)

2. **Core Network → Client App**: Real-time satellite position & signal telemetry
   - Format: JSON over WebSocket (10 Hz)
   - Fields: satellite_name, azimuth, elevation, c_n_ratio_db, eirp_dbw, path_loss_db

3. **Core Network → Web Admin**: Session events, handover logs, device provisioning status
   - Format: JSON REST API + WebSocket stream
   - Endpoints: `/api/gateways`, `/api/sessions`, `/api/handovers`, `/api/devices`, `/api/billing/*`

4. **All → Shared Infrastructure**:
   - Logging: structured JSON logs to centralized sink (ELK stack, Datadog, etc.)
   - Metrics: Prometheus format exported by each module
   - Configuration: environment variables or YAML files in shared repo

### Integration Testing Checkpoints

- **Week 3**: Orbit Calc TLE → Core Network can parse and calculate elevation angles ✓
- **Week 4**: Core Network WebSocket → Client App displays real-time antenna pointing ✓
- **Week 5**: Full system: satellite pass simulation with handover + web admin monitoring ✓

### Weekly Standup Template

**Each team reports**:
- Completed: which tasks finished this week
- Blockers: any external dependencies or technical issues
- Next: planned tasks for next week
- Risks: schedule/scope concerns

---

## Success Metrics (Project-Level)

### By End of Phase 1
- [ ] Constellation design validated (coverage ≥99.9% over Vietnam)
- [ ] Handover execution <100 ms, <0.1% packet loss (simulated)
- [ ] Client app displays live antenna tracking & signal dashboard
- [ ] Web admin shows all active sessions + recent handovers

### By End of Phase 2
- [ ] Device provisioning working (register, verify, revoke devices)
- [ ] Geo-fence enforcement active (Fixed customers restricted to bounds)
- [ ] Billing data accurate and auditable
- [ ] Web admin security page fully operational

### Demo Scenario (Project Acceptance)
**Simulate 24-hour constellation operation**:
1. Show coverage map (Vietnam fully covered)
2. Simulate satellite pass over Hanoi → Danang → HCMC
3. At each gateway transition: show handover event (client app + web admin)
4. Verify packet loss during transition <0.1%
5. Show user data usage tracker + billing impact
6. Show web admin device registry + geo-fence event log

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Orbit Calc delayed → Core Network blocked | High | Start with mock TLE; refactor integration later |
| Core Network → Client App API unstable | High | Define API contract early; use API mocking in client dev |
| Tauri/SvelteKit learning curve | Medium | Allocate 1 week for framework setup; provide tutorials |
| Handover timing requirements (100 ms) hard to achieve | High | Profile Go code early; optimize hot paths; stress-test with 100+ sessions |
| Device provisioning crypto complexity | Medium | Use Go crypto libraries (tls package); start simple (MAC-based), upgrade to hardware ID later |
| Web admin real-time updates scale to 1000+ sessions | Medium | Use efficient WebSocket serialization (binary); consider Redis pub/sub backend |

---

## Resource Allocation (Recommended)

| Phase | Team | Size | Duration | Notes |
|-------|------|------|----------|-------|
| P1 | Physics (Orbit) | 1 | 2–3 weeks | Can be part-time |
| P1 | Backend (Go) | 2 | 3–4 weeks | Critical path; may run in parallel with physics |
| P1 | Frontend (Tauri+Svelte) | 1–2 | 2–3 weeks | Depends on Core Network API stability |
| P1 | Web Admin (SvelteKit) | 2 | 3–4 weeks | Core monitoring (P1 subset) + advanced (P2) |
| P2 | Backend Extensions | 1 | 2 weeks | Device provisioning + billing |
| P2 | Client Extensions | 1 | 1.5 weeks | Geo-reporting + data usage |
| P2 | Web Admin Extensions | 1 | 1–2 weeks | Security + billing dashboards |

**Total Full-Time Equivalent (FTE)**: ~8 FTE-weeks (or 4 people for 2 weeks, distributed across teams)

---

## Optional / Out-of-Scope for MVP

- Multi-constellation support (Starlink, OneWeb comparison)
- ISL (inter-satellite link) routing
- Integration with 3GPP standards (5G NR for satellite)
- Advanced modulation (adaptive based on C/N)
- IoT/LPWAN payload (NB-IoT, LoRaWAN)
- Machine learning for traffic prediction
- Mobile iOS/Android client (web app via responsive SvelteKit is MVP)

---

## Next Steps

1. **This Week**: Finalize team assignments, set up Git repositories, establish API contract between modules
2. **Week 1**: Each team begins Phase 1 core tasks; establish weekly standups
3. **Week 3**: Checkpoint: Orbit Calc → Core Network handoff; Core Network API available to client dev
4. **Week 5**: Phase 1 integration testing begins (full constellation pass simulation)
5. **Week 6**: Phase 1 feature freeze, code review, demo preparation
6. **Week 7–10**: Phase 2 (optional) work; customer feedback & scope refinement
7. **Week 10**: Final demo + delivery package (code + architecture docs + slide deck)
