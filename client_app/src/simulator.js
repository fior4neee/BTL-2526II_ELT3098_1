const satellites = ['VNU-LEO-014', 'VNU-LEO-021', 'VNU-LEO-033', 'VNU-LEO-048'];
const gateways = ['Hanoi', 'Danang', 'HCMC'];

export function createTelemetrySample(tick) {
  const phase = tick / 18;
  const elevation = Math.max(4, 54 + Math.sin(phase) * 35);
  const azimuth = (tick * 3.8 + Math.sin(phase / 2) * 28 + 360) % 360;
  const cN = 15.5 + Math.sin(phase * 1.4) * 4.6 + Math.cos(phase / 3) * 1.8;
  const ber = Math.max(1e-8, Math.pow(10, -Math.max(4, cN / 2.2)));
  const carrierPower = -92 + cN * 1.35 + Math.sin(phase * 2) * 2;
  const handover = tick % 220 === 0 && tick > 0;

  return {
    timestamp_ms: Date.now(),
    satellite_name: satellites[Math.floor(tick / 220) % satellites.length],
    gateway_name: gateways[Math.floor(tick / 220) % gateways.length],
    azimuth_deg: azimuth,
    elevation_deg: elevation,
    beam_quality: Math.min(0.99, Math.max(0.42, 0.62 + elevation / 110 + Math.sin(phase * 1.7) * 0.08)),
    carrier_power_dbm: carrierPower,
    c_n_ratio_db: cN,
    eb_n0_db: cN - 2.1,
    ber,
    path_loss_db: 154 + (90 - elevation) * 0.09,
    eirp_dbw: 48,
    modulation_scheme: cN > 18 ? '16QAM 3/4' : cN > 13 ? 'QPSK 5/6' : 'QPSK 1/2',
    link_status: cN > 12 && elevation >= 15 ? 'connected' : cN > 9 ? 'searching' : 'outage',
    time_to_horizon_s: Math.max(0, Math.round((elevation - 4) * 22)),
    handover_active: handover,
    packet_loss_pct: handover ? 0.06 : 0.01 + Math.max(0, 13 - cN) * 0.01,
    latency_ms: 38 + Math.max(0, 18 - cN) * 2.4 + (handover ? 18 : 0),
    jitter_ms: 3 + Math.max(0, 16 - cN) * 0.7 + (handover ? 5 : 0),
    data_down_mbps: Math.max(2, 52 + Math.sin(phase * 2.4) * 22 + cN * 1.8),
    data_up_mbps: Math.max(1, 9 + Math.cos(phase * 1.8) * 4 + cN * 0.38)
  };
}

export function createLocationSample(tick) {
  const drift = Math.sin(tick / 90) * 0.012;
  return {
    timestamp_ms: Date.now(),
    latitude: 21.0278 + drift,
    longitude: 105.8342 + Math.cos(tick / 110) * 0.014,
    altitude_m: 14,
    plan_name: 'Fixed Business 300',
    plan_type: 'Fixed',
    geofence_status: Math.abs(drift) > 0.01 ? 'warning' : 'inside',
    monthly_used_gb: 812.4 + tick * 0.008,
    monthly_cap_gb: 1500,
    session_duration_s: tick,
    session_down_gb: tick * 0.00082,
    session_up_gb: tick * 0.00011,
    estimated_cost_usd: 118.5 + tick * 0.002
  };
}
