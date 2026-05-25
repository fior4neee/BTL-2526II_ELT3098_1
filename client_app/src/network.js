import { writable, get } from 'svelte/store';

const API_BASE = 'http://localhost:8081/api/v1';
const WS_BASE = 'ws://localhost:8081/api/v1';

// Stores
export const connectionState = writable('connecting'); // 'connecting' | 'connected' | 'simulation'
export const activeDevice = writable('router-vnu-leo-001');
export const telemetry = writable(null);
export const location = writable(null);
export const events = writable([]);
export const forceBreach = writable(false);

// Device Metadata matching core_network configurations
export const DEVICES = [
  { id: 'router-vnu-leo-001', name: 'Hanoi Office (Fixed)', plan: 'Fixed', home: { lat: 21.0285, lon: 105.8542, alt: 0 }, mac: '00:1B:44:11:3A:B7' },
  { id: 'router-vnu-leo-002', name: 'Hanoi Branch (Fixed)', plan: 'Fixed', home: { lat: 21.0285, lon: 105.8542, alt: 0 }, mac: '3C:A8:2A:FF:11:0C' },
  { id: 'router-vnu-leo-004', name: 'Haiphong Logistics (Mobile)', plan: 'Mobile', home: { lat: 20.8449, lon: 106.6881, alt: 0 }, mac: '54:E1:AD:99:3B:1F' },
  { id: 'router-vnu-leo-005', name: 'Can Tho Delta (Fixed)', plan: 'Fixed', home: { lat: 10.0452, lon: 105.7469, alt: 0 }, mac: 'F8:1A:67:B2:EE:90' },
  { id: 'router-vnu-leo-006', name: 'Nha Trang Station (Fixed)', plan: 'Fixed', home: { lat: 12.2388, lon: 109.1967, alt: 0 }, mac: '00:22:44:66:88:AA' },
];

const EARTH_RADIUS_KM = 6371.0;
const BOLTZMANN_DBW_PER_HZ_K = -228.6;
const ELEMENT_COUNT = 256;
const SATELLITE_EIRP_DBW = 48.0;
const ATMOSPHERIC_LOSS_DB = 1.9;
const RECEIVER_GT_DB = 11.5;
const BANDWIDTH_HZ = 25000000.0;
const BIT_RATE_BPS = 50000000.0;

// Dynamic physics calculation helpers (matching Rust antenna_tracker.rs)
function toRadians(deg) { return deg * Math.PI / 180; }
function toDegrees(rad) { return rad * 180 / Math.PI; }
function hzToDb(value) { return 10 * Math.log10(Math.max(value, 1)); }

function calculatePointing(userLat, userLon, satLat, satLon, satAlt) {
  const lat1 = toRadians(userLat);
  const lon1 = toRadians(userLon);
  const lat2 = toRadians(satLat);
  const lon2 = toRadians(satLon);
  const dLon = lon2 - lon1;

  const y = Math.sin(dLon) * Math.cos(lat2);
  const x = Math.cos(lat1) * Math.sin(lat2) - Math.sin(lat1) * Math.cos(lat2) * Math.cos(dLon);
  const azimuth = (toDegrees(Math.atan2(y, x)) + 360) % 360;

  const centralAngle = Math.acos(Math.sin(lat1) * Math.sin(lat2) + Math.cos(lat1) * Math.cos(lat2) * Math.cos(dLon));
  const radiusRatio = EARTH_RADIUS_KM / (EARTH_RADIUS_KM + satAlt);
  const elevationRad = Math.atan((Math.cos(centralAngle) - radiusRatio) / Math.max(Math.sin(centralAngle), 1e-9));
  const elevation = toDegrees(elevationRad);

  return { azimuth, elevation: Math.max(-90, Math.min(90, elevation)) };
}

function calculateSlantRange(userLat, userLon, satLat, satLon, satAlt) {
  const lat1 = toRadians(userLat);
  const lon1 = toRadians(userLon);
  const lat2 = toRadians(satLat);
  const lon2 = toRadians(satLon);
  const centralAngle = Math.acos(Math.sin(lat1) * Math.sin(lat2) + Math.cos(lat1) * Math.cos(lat2) * Math.cos(lon2 - lon1));
  const orbitalRadius = EARTH_RADIUS_KM + satAlt;
  return Math.sqrt(
    Math.pow(EARTH_RADIUS_KM, 2) +
    Math.pow(orbitalRadius, 2) -
    2 * EARTH_RADIUS_KM * orbitalRadius * Math.cos(centralAngle)
  );
}

function freeSpacePathLossDb(rangeKm, frequencyGhz = 12.0) {
  return 92.45 + 20 * Math.log10(rangeKm) + 20 * Math.log10(frequencyGhz);
}

function phasedArrayGainDb(elementCount, elevationDeg) {
  const apertureGain = 10 * Math.log10(elementCount);
  const scanLoss = -3 * (1 - Math.sin(toRadians(elevationDeg)));
  return apertureGain + Math.max(-12, scanLoss);
}

function qpskBer(ebN0Db) {
  const linear = Math.pow(10, ebN0Db / 10);
  return Math.max(1e-9, Math.min(0.5, 0.5 * Math.exp(-linear)));
}

// Haversine distance helper for local geofence alert computation
function haversineKm(lat1, lon1, lat2, lon2) {
  const dLat = toRadians(lat2 - lat1);
  const dLon = toRadians(lon2 - lon1);
  const rLat1 = toRadians(lat1);
  const rLat2 = toRadians(lat2);
  const a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
            Math.cos(rLat1) * Math.cos(rLat2) * Math.sin(dLon / 2) * Math.sin(dLon / 2);
  return 2 * EARTH_RADIUS_KM * Math.asin(Math.sqrt(a));
}

// Active session tracking & timers
let currentSession = null;
let tick = 0;
let syncInterval = null;
let ws = null;
let accumulatedBytes = 0;
let lastBytesReportTime = Date.now();

// Reset data structures on device swap
function resetSessionState() {
  currentSession = null;
  accumulatedBytes = 0;
  lastBytesReportTime = Date.now();
  telemetry.set(null);
  location.set(null);
  events.set([]);
}

// Start simulation fallback
let simTimer = null;
function startSimulationLoop() {
  if (simTimer) clearInterval(simTimer);
  simTimer = null;

  if (syncInterval) {
    clearInterval(syncInterval);
    syncInterval = null;
  }
  
  // Clean websocket connection
  if (ws) {
    ws.close();
    ws = null;
  }

  simTimer = setInterval(() => {
    tick++;
    const devId = get(activeDevice);
    const dev = DEVICES.find(d => d.id === devId) || DEVICES[0];
    
    // Simulate coordinates with optional geofence breach drift
    const breach = get(forceBreach);
    const driftOffset = breach ? 0.65 : 0; // Exceeds 50km
    
    const lat = dev.home.lat + Math.sin(tick / 90) * 0.012 + driftOffset;
    const lon = dev.home.lon + Math.cos(tick / 110) * 0.014 + driftOffset;
    
    // Compute distance
    const dist = haversineKm(dev.home.lat, dev.home.lon, lat, lon);
    const geofence_status = dev.plan === 'Mobile' ? 'inside' : 
                            dist > 50 ? 'breach' : 
                            dist > 40 ? 'warning' : 'inside';

    const phase = tick / 18;
    const elevation = Math.max(4, 54 + Math.sin(phase) * 35);
    const azimuth = (tick * 3.8 + Math.sin(phase / 2) * 28 + 360) % 360;
    const cN = 15.5 + Math.sin(phase * 1.4) * 4.6 + Math.cos(phase / 3) * 1.8;
    const ber = Math.max(1e-8, Math.pow(10, -Math.max(4, cN / 2.2)));
    const carrierPower = -92 + cN * 1.35 + Math.sin(phase * 2) * 2;
    const handover = tick % 220 === 0 && tick > 0;

    const satellite_name = ['VNU-LEO-Alpha', 'VNU-LEO-Beta', 'VNU-LEO-Gamma'][Math.floor(tick / 220) % 3];
    const gateway_name = ['Hanoi', 'Danang', 'HCMC'][Math.floor(tick / 220) % 3];

    const currentTelemetry = {
      timestamp_ms: Date.now(),
      satellite_name,
      gateway_name,
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
      link_status: geofence_status === 'breach' ? 'outage' : (cN > 12 && elevation >= 15 ? 'connected' : cN > 9 ? 'searching' : 'outage'),
      time_to_horizon_s: Math.max(0, Math.round((elevation - 4) * 22)),
      handover_active: handover,
      packet_loss_pct: handover ? 0.06 : 0.01 + Math.max(0, 13 - cN) * 0.01,
      latency_ms: 38 + Math.max(0, 18 - cN) * 2.4 + (handover ? 18 : 0),
      jitter_ms: 3 + Math.max(0, 16 - cN) * 0.7 + (handover ? 5 : 0),
      data_down_mbps: geofence_status === 'breach' ? 0 : Math.max(2, 52 + Math.sin(phase * 2.4) * 22 + cN * 1.8),
      data_up_mbps: geofence_status === 'breach' ? 0 : Math.max(1, 9 + Math.cos(phase * 1.8) * 4 + cN * 0.38)
    };

    telemetry.set(currentTelemetry);

    location.set({
      timestamp_ms: Date.now(),
      latitude: lat,
      longitude: lon,
      altitude_m: 14,
      plan_name: dev.plan === 'Mobile' ? 'Mobile Unlimited 5G' : 'Fixed Business 300',
      plan_type: dev.plan,
      geofence_status,
      monthly_used_gb: 812.4 + tick * 0.008,
      monthly_cap_gb: dev.plan === 'Mobile' ? 5000 : 1500,
      session_duration_s: tick,
      session_down_gb: tick * 0.00082,
      session_up_gb: tick * 0.00011,
      estimated_cost_usd: dev.plan === 'Mobile' ? (tick * 0.005) : (118.5 + tick * 0.002)
    });

    if (handover) {
      events.update(evs => [
        {
          id: Date.now(),
          text: `${satellite_name} via ${gateway_name}`,
          packetLoss: currentTelemetry.packet_loss_pct,
          latency: currentTelemetry.latency_ms
        },
        ...evs
      ].slice(0, 8));
    }
  }, 1000);
}

// Establish real Go backend sync loops
function startNetworkSync() {
  if (simTimer) {
    clearInterval(simTimer);
    simTimer = null;
  }

  if (syncInterval) {
    clearInterval(syncInterval);
    syncInterval = null;
  }

  // Setup WebSocket connection to capture handover/session start events
  connectWebSocket();

  // Polling loop to dynamically fetch sessions, satellites, gateways and compute metrics
  syncInterval = setInterval(async () => {
    tick++;
    const devId = get(activeDevice);
    const dev = DEVICES.find(d => d.id === devId) || DEVICES[0];

    // Compute coordinate drift
    const breach = get(forceBreach);
    const driftOffset = breach ? 0.65 : 0;
    const lat = dev.home.lat + Math.sin(tick / 90) * 0.012 + driftOffset;
    const lon = dev.home.lon + Math.cos(tick / 110) * 0.014 + driftOffset;
    const userLoc = { lat, lon, alt: 0 };

    try {
      // 1. Fetch active sessions to see if a session exists for our device
      const sessResp = await fetch(`${API_BASE}/sessions`);
      if (!sessResp.ok) throw new Error('API Error');
      const sessData = await sessResp.json();
      const activeSessions = sessData.data || [];
      let session = activeSessions.find(s => s.device_id === devId);

      // 2. If no session exists, establish connect
      if (!session) {
        const connResp = await fetch(`${API_BASE}/router/connect`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            device_id: devId,
            router_mac: dev.mac,
            location: userLoc
          })
        });

        if (connResp.status === 403) {
          // Geofence violation reported by backend
          const errData = await connResp.json();
          location.set({
            timestamp_ms: Date.now(),
            latitude: lat,
            longitude: lon,
            altitude_m: 14,
            plan_name: dev.plan === 'Mobile' ? 'Mobile Unlimited 5G' : 'Fixed Business 300',
            plan_type: dev.plan,
            geofence_status: 'breach',
            monthly_used_gb: 812.4,
            monthly_cap_gb: dev.plan === 'Mobile' ? 5000 : 1500,
            session_duration_s: 0,
            session_down_gb: 0,
            session_up_gb: 0,
            estimated_cost_usd: 0
          });
          telemetry.set({
            timestamp_ms: Date.now(),
            satellite_name: 'No Link',
            gateway_name: 'None',
            azimuth_deg: 0,
            elevation_deg: 0,
            beam_quality: 0,
            carrier_power_dbm: -120,
            c_n_ratio_db: 0,
            eb_n0_db: 0,
            ber: 0.5,
            path_loss_db: 200,
            eirp_dbw: 0,
            modulation_scheme: 'None',
            link_status: 'outage',
            time_to_horizon_s: 0,
            handover_active: false,
            packet_loss_pct: 1.0,
            latency_ms: 999,
            jitter_ms: 99,
            data_down_mbps: 0,
            data_up_mbps: 0
          });
          return;
        }

        if (connResp.ok) {
          const connData = await connResp.json();
          session = connData.session;
        }
      }

      if (!session) return;
      
      const sessionChanged = !currentSession || currentSession.session_id !== session.session_id;
      currentSession = session;

      // 3. Fetch satellites and gateways
      const [satResp, gwResp] = await Promise.all([
        fetch(`${API_BASE}/satellites`),
        fetch(`${API_BASE}/gateways`)
      ]);

      const satData = await satResp.json();
      const gwData = await gwResp.json();

      const satellitesList = satData.satellites || [];
      const gatewaysList = gwData.data || [];

      // Find the satellite the session is locked to
      const targetSat = satellitesList.find(s => s.id === session.satellite_id) || satellitesList[0];
      const targetGw = gatewaysList.find(g => g.id === session.current_gateway_id) || gatewaysList[0];

      if (!targetSat || !targetGw) return;

      // 4. Calculate actual pointing and link budget metrics using our live coordinates and satellite coordinates
      const pointing = calculatePointing(lat, lon, targetSat.location.lat, targetSat.location.lon, targetSat.location.alt);
      const range = calculateSlantRange(lat, lon, targetSat.location.lat, targetSat.location.lon, targetSat.location.alt);
      
      const pathLoss = freeSpacePathLossDb(range, 12.0);
      const antGain = phasedArrayGainDb(ELEMENT_COUNT, pointing.elevation);
      const carrierPower = SATELLITE_EIRP_DBW + antGain - pathLoss + 30; // to dBm
      
      const cN = SATELLITE_EIRP_DBW - pathLoss + RECEIVER_GT_DB - BOLTZMANN_DBW_PER_HZ_K - hzToDb(BANDWIDTH_HZ);
      const ebN0 = cN + hzToDb(BANDWIDTH_HZ) - hzToDb(BIT_RATE_BPS);
      const ber = qpskBer(ebN0);
      
      const linkStatus = cN < 9.0 ? 'outage' : (pointing.elevation < 15.0 ? 'searching' : 'connected');

      // Update telemetry
      const speedDown = linkStatus === 'connected' ? Math.max(2, 52 + cN * 1.8) : 0;
      const speedUp = linkStatus === 'connected' ? Math.max(1, 9 + cN * 0.38) : 0;

      telemetry.set({
        timestamp_ms: Date.now(),
        satellite_name: targetSat.name,
        gateway_name: targetGw.name,
        azimuth_deg: pointing.azimuth,
        elevation_deg: pointing.elevation,
        beam_quality: Math.min(0.99, Math.max(0.42, 0.48 + (pointing.elevation / 90) * 0.47)),
        carrier_power_dbm: carrierPower,
        c_n_ratio_db: cN,
        eb_n0_db: ebN0,
        ber,
        path_loss_db: pathLoss,
        eirp_dbw: SATELLITE_EIRP_DBW,
        modulation_scheme: cN > 18 ? '16QAM 3/4' : cN > 13 ? 'QPSK 5/6' : 'QPSK 1/2',
        link_status: linkStatus,
        time_to_horizon_s: Math.max(0, Math.round((pointing.elevation - 4) * 22)),
        handover_active: session.state === 'Prepare' || session.state === 'Execute',
        packet_loss_pct: session.state === 'Execute' ? 0.06 : 0.01,
        latency_ms: 38 + Math.max(0, 18 - cN) * 2.4,
        jitter_ms: 3 + Math.max(0, 16 - cN) * 0.7,
        data_down_mbps: speedDown,
        data_up_mbps: speedUp
      });

      // Update location and usage accumulation
      const bytesDown = speedDown * 1000000 * 1 / 8; // bits to bytes in 1s
      const bytesUp = speedUp * 1000000 * 1 / 8;
      const deltaBytes = bytesDown + bytesUp;
      accumulatedBytes += deltaBytes;

      const durationS = Math.floor((Date.now() - new Date(session.start_time).getTime()) / 1000);

      location.set({
        timestamp_ms: Date.now(),
        latitude: lat,
        longitude: lon,
        altitude_m: 14,
        plan_name: dev.plan === 'Mobile' ? 'Mobile Unlimited 5G' : 'Fixed Business 300',
        plan_type: dev.plan,
        geofence_status: 'inside', // will be overridden in connect error if breach
        monthly_used_gb: 812.4 + (accumulatedBytes / 1024 / 1024 / 1024),
        monthly_cap_gb: dev.plan === 'Mobile' ? 5000 : 1500,
        session_duration_s: durationS,
        session_down_gb: (bytesDown * tick) / 1024 / 1024 / 1024,
        session_up_gb: (bytesUp * tick) / 1024 / 1024 / 1024,
        estimated_cost_usd: dev.plan === 'Mobile' ? (tick * 0.005) : (118.5 + tick * 0.002)
      });

      // 5. Periodically send billing usage events to Go server (every 5 seconds)
      const now = Date.now();
      if (now - lastBytesReportTime >= 5000 && deltaBytes > 0) {
        fetch(`${API_BASE}/billing/event`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            event_id: `evt-${Date.now()}-${Math.floor(Math.random()*10000)}`,
            type: 'usage',
            session_id: session.session_id,
            device_id: devId,
            bytes_delta: Math.round(accumulatedBytes),
            ts: new Date().toISOString()
          })
        }).catch(err => console.error('Failed to post billing event:', err));
        
        lastBytesReportTime = now;
      }

    } catch (err) {
      console.error('Connection to Go backend failed, falling back to Simulation Mode:', err);
      connectionState.set('simulation');
      startSimulationLoop();
    }
  }, 1000);
}

function connectWebSocket() {
  if (ws) ws.close();

  // Short lived auth token parameter as requested by Go backend
  ws = new WebSocket(`${WS_BASE}/telemetry/stream?token=vnu-leo-token-client`);

  ws.onopen = () => {
    console.log('🔌 WebSocket stream established with Core Network.');
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (data.event === 'handover_execute') {
        events.update(evs => [
          {
            id: Date.now(),
            text: `Handover Execute: Success`,
            packetLoss: 0.06,
            latency: 48.0
          },
          ...evs
        ].slice(0, 8));
      } else if (data.event === 'session_start') {
        events.update(evs => [
          {
            id: Date.now(),
            text: `Session started: ${data.session_id}`,
            packetLoss: 0,
            latency: 38.0
          },
          ...evs
        ].slice(0, 8));
      }
    } catch (e) {
      console.error('Failed parsing WS message:', e);
    }
  };

  ws.onerror = (e) => {
    console.error('WS Connection error:', e);
  };

  ws.onclose = () => {
    console.log('🔌 WebSocket stream disconnected.');
  };
}

// Check network health and select mode
export async function initializeNetwork() {
  resetSessionState();
  connectionState.set('connecting');

  try {
    const res = await fetch(`${API_BASE}/health`);
    if (res.ok) {
      connectionState.set('connected');
      startNetworkSync();
    } else {
      connectionState.set('simulation');
      startSimulationLoop();
    }
  } catch (err) {
    console.log('Go server health check failed. Starting in Simulation Fallback Mode.');
    connectionState.set('simulation');
    startSimulationLoop();
  }
}

// Watch device ID changes to reconnect sessions
activeDevice.subscribe((devId) => {
  resetSessionState();
  const state = get(connectionState);
  if (state === 'connected') {
    startNetworkSync();
  } else if (state === 'simulation') {
    startSimulationLoop();
  }
});

// Watch geofence breach toggle
forceBreach.subscribe(() => {
  const state = get(connectionState);
  if (state === 'connected') {
    startNetworkSync();
  }
});
