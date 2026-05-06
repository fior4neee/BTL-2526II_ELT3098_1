# Gateway & Handover (Core Network) - Remediation List

**Module Owner**: Core Network/Backend Team  
**Phase**: P1 (Core) + P2 (Optional)  
**Requirement**: Bài toán Gateway (Core Network)

## Current State
- `bai2/handover.py` contains a basic elevation-angle comparison for three gateways.
- No session management, connection state machine, or multi-gateway coordination.
- No handover metrics, packet loss tracking, or service continuity.

## Phase 1 Deliverables (Core)

### 1. **gateway_nodes.go**
- [ ] Define `Gateway` struct representing gateway stations (Hanoi, Danang, HCMC):
  - Location (latitude, longitude, altitude)
  - Capacity (bandwidth, max concurrent sessions)
  - Signal threshold (minimum elevation for link viability)
  - Antenna model (gain, beam width)
- [ ] Implement `GatewayPool` to manage all gateways:
  - Elevation angle calculation from satellite ephemeris
  - Link quality assessment (C/N ratio, SNR)
  - Availability check (above/below minimum elevation)
- [ ] Track gateway state: alive/dead, load, recent handover events
- [ ] Logging/telemetry: connectivity events, signal metrics per satellite

### 2. **handover_manager.go**
- [ ] Implement `HandoverManager` state machine:
  - **Acquire**: Router connects to satellite → best gateway elected by elevation/signal strength
  - **Hold**: Maintain connection while satellite passes overhead
  - **Prepare**: Detect upcoming handover (satellite approaching horizon, new gateway becoming viable)
  - **Execute**: Switch session from old to new gateway with minimal packet loss
  - **Release**: Clean up session at old gateway
- [ ] Handover algorithm:
  - Hysteresis: avoid rapid switching (dead band: ±2° elevation)
  - Predictive: use satellite ephemeris to pre-establish session at next gateway
  - Transparent: re-route in-flight packets to new gateway without dropping
- [ ] Session management:
  - Maintain connection state across handovers (TCP stateful, UDP best-effort)
  - Track sequence numbers/ACKs for packet loss detection
  - Support both stateful (Voice/IP) and stateless (bulk data) flows
- [ ] Metrics:
  - Handover duration (target: <100 ms)
  - Packet loss during transition
  - Session continuity success rate
  - Redundancy: if 2+ gateways are viable, load-balance or maintain hot-standby

### 3. **Integration with Orbit Calc**
- [ ] Accept satellite ephemeris (TLE or SGP4 state vectors) from `orbit_calc` module
- [ ] Subscribe to satellite position updates and re-calculate elevation angles in real-time
- [ ] Trigger handover logic when elevation crosses threshold

## Phase 2 Enhancements (Optional)

### 4. **device_provisioning.go**
- [ ] Implement device registration & verification:
  - MAC address registration for each router
  - Hardware ID (secure enclave) verification to prevent spoofing
  - Certificate-based authentication (device certificate signed by ISP)
- [ ] Provisioning workflow:
  - New router registers with ISP, receives provisioning code
  - Gateway validates certificate and MAC during first connection
  - Reject unregistered/spoofed devices
- [ ] Revocation: support blacklisting of compromised devices
- [ ] Audit trail: log all device connections, registration, revocations

### 5. **spaciotemporal_billing.go**
- [ ] Define subscription models:
  - **Fixed (Geo-fenced)**: router locked to geographic region (e.g., single city); service disabled if moved >50 km
  - **Mobile**: router can connect anywhere; billed per GB or time-based
- [ ] Implement geo-fence checker:
  - On each handover, verify router's last-known location
  - If Fixed subscription and location drifts beyond geo-fence, trigger warning/disconnect
- [ ] Billing engine:
  - Track data usage per session
  - Attribute usage to subscription tier
  - Log billing events for ISP accounting
- [ ] Admin API: override geo-fencing (e.g., for emergency services)

## Testing & Validation (P1)

- [ ] Unit tests for elevation calculation, handover state transitions
- [ ] Integration test: simulate satellite passing over 3 gateways, track handover events
- [ ] Performance test: stress-test with 100+ concurrent sessions, measure CPU/memory
- [ ] Failure mode: test behavior when one gateway goes offline
- [ ] Demo script: show a simulated 24-hour period with all handover events logged

## References
- 3GPP handover specifications (LTE/5G NSA)
- ITU-R P.618 (path loss & propagation)
- RFC 5118 (Mobile IPv6 Handover Optimization)
