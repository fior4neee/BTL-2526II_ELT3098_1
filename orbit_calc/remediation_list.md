# Orbit & Coverage Calculation - Remediation List

**Module Owner**: Orbit/Physics Team  
**Phase**: P1 (Core)  
**Requirement**: Bài toán Quỹ đạo & Phủ sóng

## Current State
- `bai1/LEO.py` contains a generic constellation calculator (heuristic).
- `bai1/vn_internet_voip.py` applies a 0.4 multiplier for Vietnam but lacks rigor.
- No simulation, validation, or constellation geometry implemented.

## Phase 1 Deliverables (Core)

### 1. **coverage_calculator.py**
- [ ] Implement `VietnamCoverageAnalyzer` class that validates continuous coverage over Vietnam (8°N–24°N).
- [ ] Calculate required satellite count based on:
  - Orbit altitude (LEO 500–1200 km)
  - Minimum elevation angle (≥15° for usable signal)
  - Revisit time per ground point (≤15 min for real-time Internet/VoIP)
  - Walker Delta constellation model (inclination, number of planes, satellites per plane)
- [ ] Separate calculations for two use cases:
  - **Internet/VoIP**: Low altitude (~500 km), high revisit frequency, lower latency
  - **Data/Weather**: Higher altitude (~1000 km), longer dwell time per pass, higher throughput
- [ ] Output: constellation parameters (N satellites, planes, inclination, RAAN spacing, mean motion)
- [ ] Validate that proposed constellation covers all of Vietnam at all times with min elevation ≥15°.

### 2. **traffic_model.py**
- [ ] Define `TrafficModel` class to characterize service requirements:
  - Internet/VoIP: latency budget (≤50 ms), minimum availability (≥99%), bitrate (1–10 Mbps per user)
  - Data/Radar: throughput (≥100 Mbps during passes), burst capacity, data volume per pass
- [ ] Calculate link budget (EIRP, G/T, path loss, atmospheric loss, C/N) for both services
- [ ] Determine minimum Eb/N0 and coding scheme required
- [ ] Model on-ground dwell time: how long satellite remains above minimum elevation for given latitude
- [ ] Output: traffic demand vs. available capacity per satellite pass

### 3. **simulation.ipynb** (Jupyter Notebook - P1)
- [ ] Load SGP4 orbital elements or generate analytical TLE for proposed Walker constellation
- [ ] Simulate satellite positions over 24 hours for Vietnamese ground points (Hanoi, Danang, HCMC + grid of 5×5 sites)
- [ ] Plot:
  - Ground track of satellites
  - Coverage footprint (showing elevation contours)
  - Handover zones (where coverage switches between satellites)
  - Revisit timeline for each ground point
- [ ] Verify:
  - Zero coverage gaps in Vietnam airspace
  - Handover locations and timing
  - Latency variation (Doppler, propagation delay)
- [ ] Export results: CSV of coverage events, constellation TLE, coverage statistics

## Phase 2 Enhancements (Optional)

- Multi-constellation analysis (compare different Walker parameters)
- Atmospheric modeling (rain attenuation, ionospheric delay)
- Collision avoidance and orbital debris constraints
- Inter-satellite link (ISL) topology and routing feasibility

## References
- Walker Delta constellation design paper
- Satellite link budget calculation (ITU-R P.618)
- NORAD TLE format and SGP4 propagation
