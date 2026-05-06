# VNU-LEO System Architecture

## System Overview

VNU-LEO is a Low-Earth-Orbit (LEO) satellite constellation system serving Vietnam with two primary services:
- **Internet/VoIP**: low-latency connectivity
- **Data/Weather**: high-throughput data imaging and radar

The system consists of three main components:

1. **Orbit & Coverage Calculation** (`orbit_calc/`): Constellation design and coverage simulation
2. **Gateway Core Network** (`core_network/`): Satellite handover, session management, and billing
3. **End-User Router** (`client_app/`): Desktop application for signal monitoring and usage tracking
4. **ISP Web Admin** (`web_admin/`): Operations dashboard for infrastructure monitoring

## Module Interaction

```
┌─────────────────────────────────────────────────────────────────┐
│                         Orbit Calculation                       │
│  (Coverage Simulator, Traffic Model, Constellation Design)      │
│  Output: Satellite TLE, Coverage Maps, Handover Zones           │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 ├──> Ephemeris Feed (Real-time Satellite Positions)
                 │
┌────────────────▼────────────────────────────────────────────────┐
│                   Gateway Core Network (Go)                     │
│  - Elevation Angle Calculation (from Ephemeris)                 │
│  - Handover Manager (State Machine)                             │
│  - Session Management (TCP/UDP)                                 │
│  - Device Provisioning & Verification (P2)                      │
│  - Spatio-temporal Billing (P2)                                 │
│  Output: Session Events, Handover Logs, Telemetry              │
└────────────────┬────────────────────────────────────────────────┘
                 │
        ┌────────┴─────────────────────┐
        │                              │
┌───────▼──────────────────┐   ┌──────▼──────────────────┐
│   End-User Router App    │   │   ISP Web Admin         │
│   (Tauri + Svelte)       │   │   (SvelteKit)           │
│ - Antenna Tracking       │   │ - Session Monitoring    │
│ - Signal Dashboard       │   │ - Handover History      │
│ - Data Usage Tracking    │   │ - Device Security (P2)  │
│ - Geo-fencing (P2)       │   │ - Billing Dashboard (P2)│
└──────────────────────────┘   └─────────────────────────┘
```

## Data Flow

### Real-time Telemetry
1. **Orbit Calc** computes satellite positions (SGP4 propagation)
2. **Core Network** receives positions, calculates elevation angles to each gateway
3. **Gateway** broadcasts signal quality (C/N, BER) to active sessions
4. **End-User Router** receives signal metrics, updates antenna pointing and dashboard display
5. **Web Admin** aggregates telemetry from all gateways for operator view

### Session Lifecycle (Handover)
1. **Router** initiates connection; core network identifies best gateway
2. **Hold Phase**: session maintained while satellite above minimum elevation
3. **Handover Preparation**: next gateway pre-provisioned before current satellite horizon
4. **Switch Event**: packets redirected to new gateway with minimal loss (<100 ms target)
5. **Cleanup**: old gateway releases session; router acknowledges new gateway

### Billing Flow (P2)
1. **End-User Router** reports location (GPS) and data usage to core network
2. **Spatio-temporal Billing** module checks subscription (Fixed vs. Mobile)
3. **Geo-fence enforcement**: if Fixed customer outside bounds → disconnect + alert to web admin
4. **Billing events**: logged for ISP accounting and reconciliation

## File Layout

```
BTL-2526II_ELT3098_1/
├── orbit_calc/
│   ├── coverage_calculator.py         (P1: Coverage analysis)
│   ├── traffic_model.py               (P1: Link budget & revisit)
│   ├── simulation.ipynb               (P1: Visualization & validation)
│   └── remediation_list.md
├── core_network/
│   ├── gateway_nodes.go               (P1: Gateway infrastructure)
│   ├── handover_manager.go            (P1: Handover state machine)
│   ├── device_provisioning.go         (P2: Device verification)
│   ├── spaciotemporal_billing.go      (P2: Billing & geo-fence)
│   └── remediation_list.md
├── client_app/
│   ├── src-tauri/
│   │   ├── attenna_tracker.rs         (P1: Antenna & signal calc)
│   │   └── geo_reporter.rs            (P2: GPS & geo-fence)
│   ├── src/
│   │   ├── SignalDashboard.svelte     (P1: Real-time UI)
│   │   └── DataUsage.svelte           (P2: Usage tracking)
│   └── remediation_list.md
├── web_admin/
│   ├── src/routes/
│   │   ├── +page.svelte               (P1: Overview dashboard)
│   │   ├── monitoring/
│   │   │   └── +page.svelte           (P1: Detailed monitoring)
│   │   └── security/
│   │       └── +page.svelte           (P2: Device & billing)
│   └── remediation_list.md
├── docs/
│   ├── architecture.md                (this file)
│   └── BaiTapNhom.md                  (original spec)
└── temp/
    ├── bai1/                          (legacy orbit code)
    ├── bai2/                          (legacy handover code)
    └── starsim_notebook.ipynb         (legacy notebook)
```

## Technology Stack

| Component | Language | Framework | Purpose |
|-----------|----------|-----------|---------|
| Orbit Calc | Python | NumPy, Poliastro, Matplotlib, Jupyter | Science & simulation |
| Core Network | Go | Standard library + gRPC (optional) | High-performance backend |
| Client App | Rust (backend), Svelte (frontend) | Tauri v2 | Desktop application |
| Web Admin | TypeScript, Svelte | SvelteKit, TailwindCSS | Web dashboard |

## Deployment Considerations

### Development
- Each module can be developed and tested independently
- Orbit calc runs locally (Python Jupyter)
- Core network runs on simulated gateway (localhost or Docker)
- Client app runs as Tauri desktop binary
- Web admin runs via SvelteKit dev server

### Production
- Orbit calc: compute satellite ephemeris once per day, cache results
- Core network: deployed on edge/cloud with high availability (3+ instances per gateway)
- Client app: distributed as Tauri binary (Windows/macOS/Linux)
- Web admin: deployed on secure server with reverse proxy (nginx/caddy), TLS certificate

### Integration Points
- Core network ↔ Orbit calc: ephemeris via REST API or file import
- End-user router ↔ Core network: WebSocket for real-time telemetry, REST for provisioning
- Web admin ↔ Core network: REST API + WebSocket for monitoring
- All services: structured logging (JSON), metrics export (Prometheus format for monitoring)

## Security & Compliance

- **Device Verification (P2)**: MAC/Hardware ID + certificate-based authentication
- **Spatio-temporal Billing (P2)**: geo-fence enforcement with audit trail
- **Admin Access**: role-based (Super Admin, Operator, Billing Manager)
- **Encryption**: TLS 1.3 for all network traffic, encrypted storage for device keys
- **Logging**: immutable audit logs for compliance (data retention: 90 days minimum)

## Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Handover duration | <100 ms | Switch between gateways |
| Packet loss during handover | <0.1% | Voice quality SLA |
| Antenna tracking latency | <500 ms | UI responsiveness |
| Satellite position update rate | 10 Hz | Real-time visual smoothness |
| Gateway API response time | <100 ms | Session state queries |
| Coverage continuity | 99.9% | No gaps in Vietnam airspace |
| Session availability | 99% | SLA per subscription tier |

## Future Enhancements (Beyond P2)

1. Inter-satellite links (ISL) for backbone routing
2. Multi-constellation support (LEO + GEO hybrid)
3. Machine learning for traffic prediction and resource optimization
4. Advanced modulation (adaptive based on C/N ratio)
5. Integration with 3GPP standards (LTE NTN, 5G NR for satellite)
6. IoT/LPWAN support (NB-IoT, LoRaWAN via satellite)
