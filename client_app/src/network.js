import { writable, get } from 'svelte/store';

const API_BASE = '/api/v1';
const WS_BASE = `ws://${window.location.host}/api/v1`;

// Stores
export const connectionState = writable('connecting'); // 'connecting' | 'connected' | 'error'
const urlDevice = typeof window !== 'undefined'
  ? new URLSearchParams(window.location.search).get('device_id')
  : null;
const envDevice = typeof import.meta !== 'undefined' ? import.meta.env?.VITE_DEVICE_ID : null;
const defaultDeviceId = envDevice || urlDevice || 'router-vnu-leo-001';

export const activeDevice = writable(defaultDeviceId);
export const telemetry = writable(null);
export const location = writable(null);
export const events = writable([]);
export const forceBreach = writable(false);
export const deviceStatus = writable('unknown');

// Device Metadata matching core_network configurations
export const DEVICES = [
  { id: 'router-vnu-leo-001', name: 'Hanoi Office (Fixed)', plan: 'Fixed', home: { lat: 21.0285, lon: 105.8542, alt: 0 }, mac: '00:1B:44:11:3A:B7' },
  { id: 'router-vnu-leo-002', name: 'Hanoi Branch (Fixed)', plan: 'Fixed', home: { lat: 21.0285, lon: 105.8542, alt: 0 }, mac: '3C:A8:2A:FF:11:0C' },
  { id: 'router-vnu-leo-003', name: 'Danang Hub (Fixed)', plan: 'Fixed', home: { lat: 16.0470, lon: 108.2062, alt: 0 }, mac: '12:34:56:78:9A:BC' },
  { id: 'router-vnu-leo-004', name: 'Haiphong Logistics (Mobile)', plan: 'Mobile', home: { lat: 20.8449, lon: 106.6881, alt: 0 }, mac: '54:E1:AD:99:3B:1F' },
  { id: 'router-vnu-leo-005', name: 'HCMC Headquarters (Fixed)', plan: 'Fixed', home: { lat: 10.7626, lon: 106.6601, alt: 0 }, mac: 'F8:1A:67:B2:EE:90' },
  { id: 'router-vnu-leo-006', name: 'Nha Trang Station (Fixed)', plan: 'Fixed', home: { lat: 12.2388, lon: 109.1967, alt: 0 }, mac: '00:22:44:66:88:AA' },
  { id: 'router-vnu-leo-007', name: 'Liams Phone (Mobile)', plan: 'Mobile', home: { lat: 21.0285, lon: 105.8542, alt: 0 }, mac: 'AA:BB:CC:DD:EE:FF' },
  { id: 'router-vnu-leo-008', name: 'Test Router (Mobile)', plan: 'Mobile', home: { lat: 16.0470, lon: 108.2062, alt: 0 }, mac: '98:76:54:32:10:FE' },
  { id: 'router-vnu-leo-009', name: 'Stolen Device 009 (Mobile)', plan: 'Mobile', home: { lat: 21.0285, lon: 105.8542, alt: 0 }, mac: 'FE:DC:BA:98:76:54' },
  { id: 'router-vnu-leo-010', name: 'Unpaid Bill 010 (Fixed)', plan: 'Fixed', home: { lat: 10.7626, lon: 106.6601, alt: 0 }, mac: '1A:2B:3C:4D:5E:6F' },
  { id: 'router-vnu-leo-011', name: 'Unknown Device 011 (Mobile)', plan: 'Mobile', home: { lat: 21.0285, lon: 105.8542, alt: 0 }, mac: '00:AA:BB:CC:DD:EE' },
  { id: 'router-vnu-leo-014', name: 'Compromised Router 014 (Fixed)', plan: 'Fixed', home: { lat: 16.0470, lon: 108.2062, alt: 0 }, mac: '01:03:05:07:09:0B' }
];

const EARTH_RADIUS_KM = 6371.0;

// Geo helpers
function toRadians(deg) { return deg * Math.PI / 180; }

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

function getDeviceConfig(devId) {
  return DEVICES.find(d => d.id === devId) || DEVICES[0];
}

function buildDeviceLocation(dev, tickValue, breach) {
  const driftOffset = breach ? 0.65 : 0;
  const lat = dev.home.lat + Math.sin(tickValue / 90) * 0.012 + driftOffset;
  const lon = dev.home.lon + Math.cos(tickValue / 110) * 0.014 + driftOffset;
  return { lat, lon, alt: 0 };
}

function geofenceStatus(dev, loc) {
  if (dev.plan === 'Mobile') return 'inside';
  const dist = haversineKm(dev.home.lat, dev.home.lon, loc.lat, loc.lon);
  if (dist > 50) return 'breach';
  if (dist > 40) return 'warning';
  return 'inside';
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
  deviceStatus.set('unknown');
}

// Establish real Go backend sync loops
function startNetworkSync() {
  if (syncInterval) {
    clearInterval(syncInterval);
    syncInterval = null;
  }

  // Setup WebSocket connection to capture telemetry
  connectWebSocket();

  // Polling loop to ensure session exists and push location updates
  syncInterval = setInterval(async () => {
    tick++;
    const devId = get(activeDevice);
    const dev = getDeviceConfig(devId);
    const breach = get(forceBreach);
    const userLoc = buildDeviceLocation(dev, tick, breach);

    try {
      const sessResp = await fetch(`${API_BASE}/sessions`);
      if (!sessResp.ok) throw new Error('API Error');
      const sessData = await sessResp.json();
      const activeSessions = sessData.sessions || sessData.data || [];
      let session = activeSessions.find(s => s.device_id === devId);

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
          const errData = await connResp.json();
          deviceStatus.set(errData.device_status || 'blocked');
          location.set({
            timestamp_ms: Date.now(),
            latitude: userLoc.lat,
            longitude: userLoc.lon,
            altitude_m: 14,
            plan_name: dev.plan === 'Mobile' ? 'Mobile Unlimited 5G' : 'Fixed Business 300',
            plan_type: dev.plan,
            geofence_status: geofenceStatus(dev, userLoc),
            monthly_used_gb: 0,
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
      currentSession = session;

      const updateResp = await fetch(`${API_BASE}/router/update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ device_id: devId, location: userLoc })
      });

      if (updateResp.status === 403) {
        const errData = await updateResp.json();
        deviceStatus.set(errData.device_status || 'blocked');
      }

      location.set({
        timestamp_ms: Date.now(),
        latitude: userLoc.lat,
        longitude: userLoc.lon,
        altitude_m: 14,
        plan_name: dev.plan === 'Mobile' ? 'Mobile Unlimited 5G' : 'Fixed Business 300',
        plan_type: dev.plan,
        geofence_status: geofenceStatus(dev, userLoc),
        monthly_used_gb: accumulatedBytes / 1024 / 1024 / 1024,
        monthly_cap_gb: dev.plan === 'Mobile' ? 5000 : 1500,
        session_duration_s: Math.floor((Date.now() - new Date(session.start_time).getTime()) / 1000),
        session_down_gb: accumulatedBytes / 1024 / 1024 / 1024,
        session_up_gb: accumulatedBytes / 1024 / 1024 / 1024,
        estimated_cost_usd: 0
      });
    } catch (err) {
      console.error('Connection to Go backend failed:', err);
      connectionState.set('error');
    }
  }, 1000);
}

function connectWebSocket() {
  if (ws) ws.close();

  // Short lived auth token parameter as requested by Go backend
  ws = new WebSocket(`${WS_BASE}/telemetry/stream?token=vnu-leo-token-client&topics=telemetry,handovers,sessions,satellites`);

  ws.onopen = () => {
    console.log('🔌 WebSocket stream established with Core Network.');
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      const devId = get(activeDevice);
      if (Array.isArray(data.telemetry)) {
        const sample = data.telemetry.find(t => t.device_id === devId);
        if (sample) {
          deviceStatus.set(sample.device_status || 'active');
          const visualSat = calculateWebAdminSat(sample.gateway_id);
          const isNoLink = !visualSat;
          telemetry.set({
            timestamp_ms: Date.now(),
            satellite_name: visualSat || 'No Link',
            gateway_name: sample.gateway_name,
            azimuth_deg: sample.azimuth_deg,
            elevation_deg: isNoLink ? 0 : sample.elevation_deg,
            beam_quality: isNoLink ? 0 : sample.beam_quality,
            carrier_power_dbm: isNoLink ? -120 : sample.carrier_power_dbm,
            c_n_ratio_db: isNoLink ? 0 : sample.c_n_ratio_db,
            eb_n0_db: isNoLink ? 0 : sample.eb_n0_db,
            ber: isNoLink ? 1e-1 : sample.ber,
            path_loss_db: sample.path_loss_db,
            eirp_dbw: sample.eirp_dbw,
            modulation_scheme: isNoLink ? 'None' : sample.modulation_scheme,
            link_status: isNoLink ? 'outage' : sample.link_status,
            time_to_horizon_s: 0,
            handover_active: sample.handover_active,
            packet_loss_pct: isNoLink ? 0 : sample.packet_loss_pct,
            latency_ms: isNoLink ? 0 : sample.latency_ms,
            jitter_ms: sample.jitter_ms,
            data_down_mbps: sample.data_down_mbps,
            data_up_mbps: sample.data_up_mbps
          });

          if (sample.location) {
            const dev = getDeviceConfig(devId);
            const fence = geofenceStatus(dev, sample.location);
            const bytesDown = sample.data_down_mbps * 1_000_000 / 8 / 5;
            const bytesUp = sample.data_up_mbps * 1_000_000 / 8 / 5;
            accumulatedBytes += bytesDown + bytesUp;
            location.set({
              timestamp_ms: Date.now(),
              latitude: sample.location.lat,
              longitude: sample.location.lon,
              altitude_m: sample.location.alt * 1000,
              plan_name: dev.plan === 'Mobile' ? 'Mobile Unlimited 5G' : 'Fixed Business 300',
              plan_type: dev.plan,
              geofence_status: fence,
              monthly_used_gb: accumulatedBytes / 1024 / 1024 / 1024,
              monthly_cap_gb: dev.plan === 'Mobile' ? 5000 : 1500,
              session_duration_s: currentSession ? Math.floor((Date.now() - new Date(currentSession.start_time).getTime()) / 1000) : 0,
              session_down_gb: accumulatedBytes / 1024 / 1024 / 1024,
              session_up_gb: accumulatedBytes / 1024 / 1024 / 1024,
              estimated_cost_usd: 0
            });
          }
        }
      }

      if (Array.isArray(data.handovers) && currentSession) {
        const recent = data.handovers.filter(h => h.session_id === currentSession.session_id).slice(-1)[0];
        if (recent) {
          events.update(evs => [
            {
              id: Date.now(),
              text: `${recent.source_gateway_id} → ${recent.target_gateway_id}`,
              packetLoss: recent.packet_loss_pct ?? 0,
              latency: get(telemetry)?.latency_ms ?? 0
            },
            ...evs
          ].slice(0, 8));
        }
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
      connectionState.set('error');
    }
  } catch (err) {
    console.log('Go server health check failed.');
    connectionState.set('error');
  }
}

// Watch device ID changes to reconnect sessions
activeDevice.subscribe((devId) => {
  resetSessionState();
  const state = get(connectionState);
  if (state === 'connected') {
    startNetworkSync();
  }
});

// Watch geofence breach toggle
forceBreach.subscribe(() => {
  const state = get(connectionState);
  if (state === 'connected') {
    startNetworkSync();
  }
});

// --- Shim to match Web Admin's visual satellite connections ---
function calculateWebAdminSat(gwId) {
  const STATIC_GW = {
    'GW-HAN-01': { lat: 21.028, lng: 105.854 },
    'GW-DAN-01': { lat: 16.047, lng: 108.206 },
    'GW-HCM-01': { lat: 10.763, lng: 106.660 }
  };
  const gw = STATIC_GW[gwId];
  if (!gw) return null;
  
  const t = Date.now() / 1000;
  const N_INT = (15.24308387 * 2 * Math.PI) / 86400;
  const EARTH_ROT = 7.2921150e-5;
  const INC = 53 * Math.PI / 180;
  const D2R = Math.PI / 180;
  
  let best = null, bestEl = -Infinity;
  
  const testSatsWalker = (N, P, F, nRate, altKm, prefix) => {
    const S = N / P;
    const raanStep = 360 / P;
    const m0Step = 360 / S;
    const phaseShift = (F * 360) / N;
    let count = 1;

    for (let p = 0; p < P; p++) {
      const raan = p * raanStep * D2R;
      for (let s = 0; s < S; s++) {
        const m0Deg = (s * m0Step + p * phaseShift) % 360;
        const m0 = m0Deg * D2R;
        const M = m0 + nRate * t;
        const xOrb = Math.cos(M), yOrb = Math.sin(M);
        const x3 = xOrb, y3 = yOrb * Math.cos(INC), z3 = yOrb * Math.sin(INC);
        const xEci = x3 * Math.cos(raan) - y3 * Math.sin(raan);
        const yEci = x3 * Math.sin(raan) + y3 * Math.cos(raan);
        const theta = EARTH_ROT * t;
        const xEcef =  xEci * Math.cos(theta) + yEci * Math.sin(theta);
        const yEcef = -xEci * Math.sin(theta) + yEci * Math.cos(theta);
        let lon = Math.atan2(yEcef, xEcef) * (180 / Math.PI);
        lon = ((lon + 540) % 360) - 180;
        const lat = Math.asin(Math.max(-1, Math.min(1, z3))) * (180 / Math.PI);
        
        const cosEta = Math.sin(gw.lat*D2R) * Math.sin(lat*D2R)
                     + Math.cos(gw.lat*D2R) * Math.cos(lat*D2R)
                     * Math.cos((lon - gw.lng)*D2R);
        const c = Math.max(-1, Math.min(1, cosEta));
        const a = 6371 + altKm;
        const dist = Math.sqrt(a*a - 2*a*6371*c + 6371*6371);
        const suffix = count.toString().padStart(3, '0');
        count++;

        if (dist < 1e-6) {
           bestEl = 90; best = `${prefix}${suffix}`;
           continue;
        }
        const sinEl = (a * c - 6371) / dist;
        const el = Math.asin(Math.max(-1, Math.min(1, sinEl))) * (180 / Math.PI);
        if (el >= 15 && el > bestEl) { bestEl = el; best = `${prefix}${suffix}`; }
      }
    }
  };

  testSatsWalker(306, 17, 0, N_INT, 500, 'I-VNU-LEO-');
  
  return best;
}
