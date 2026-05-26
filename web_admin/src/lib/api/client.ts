import type {
  Device,
  Gateway,
  HandoverEvent,
  HandoverTriggerResponse,
  Satellite,
  SecurityAlert,
  Session,
} from '$lib/types';

const API_BASE = (import.meta.env.VITE_CORE_API_BASE ?? '/api/v1').replace(/\/$/, '');
const DEVICES_ENABLED = import.meta.env.VITE_ENABLE_DEVICES === 'true';

function resolveApiUrl(path: string): string {
  if (path.startsWith('http')) return path;
  if (typeof window === 'undefined') {
    return `${API_BASE}${path}`;
  }
  const base = API_BASE.startsWith('http')
    ? API_BASE
    : new URL(API_BASE, window.location.origin).toString();
  return `${base}${path}`;
}

async function requestJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(resolveApiUrl(path), {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  });

  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`;
    try {
      const body = await res.json();
      message = body.error ?? message;
    } catch {
      // Keep HTTP status as the fallback error.
    }
    throw new Error(message);
  }

  return res.json() as Promise<T>;
}

function gatewayStatus(value: string): Gateway['status'] {
  if (value === 'degraded' || value === 'dead') return value;
  return 'alive';
}

export function normalizeGateway(raw: any): Gateway {
  return {
    id: raw.id,
    name: raw.name ?? raw.id,
    lat: raw.location?.lat ?? raw.lat ?? 0,
    lng: raw.location?.lon ?? raw.lng ?? 0,
    altitudeKm: raw.location?.alt ?? raw.altitude_km ?? 0,
    status: gatewayStatus(raw.status),
    currentSessions: raw.current_sessions ?? raw.currentSessions ?? 0,
    maxSessions: raw.max_sessions ?? raw.maxSessions ?? 0,
    minElevationDeg: raw.min_elevation_deg ?? raw.minElevationDeg ?? 0,
    antennaGainDbi: raw.antenna?.gain_dbi ?? raw.antennaGainDbi ?? 0,
    beamWidthDeg: raw.antenna?.beam_width_deg ?? raw.beamWidthDeg ?? 0,
    cpuPct: raw.cpu_pct ?? raw.cpuPct,
    memoryPct: raw.memory_pct ?? raw.memoryPct,
    bandwidthMbps: raw.bandwidth_mbps ?? raw.bandwidthMbps,
    uptimePct: raw.uptime_pct ?? raw.uptimePct,
    lastSeen: raw.last_seen ? new Date(raw.last_seen) : undefined,
  };
}

export function normalizeSession(raw: any): Session {
  return {
    id: raw.session_id ?? raw.id,
    routerMac: raw.router_mac ?? raw.routerMac,
    satelliteId: raw.satellite_id ?? raw.satelliteId ?? 'n/a',
    currentGatewayId: raw.current_gateway_id ?? raw.currentGatewayId,
    nextGatewayId: raw.next_gateway_id ?? raw.nextGatewayId ?? undefined,
    state: raw.state,
    startTime: new Date(raw.start_time ?? raw.startTime ?? Date.now()),
    dataMb: raw.data_mb ?? raw.dataMb,
    cnRatioDb: raw.cn_ratio_db ?? raw.cnRatioDb,
    lastActivity: raw.last_activity ? new Date(raw.last_activity) : undefined,
  };
}

export function normalizeSatellite(raw: any): Satellite {
  const location = raw.location ?? {};
  return {
    satelliteId: raw.satellite_id ?? raw.id,
    latitude: raw.latitude ?? location.lat ?? 0,
    longitude: raw.longitude ?? location.lon ?? 0,
    altitude: raw.altitude ?? location.alt ?? 0,
    elevation: raw.elevation ?? raw.elevation_deg ?? 0,
    azimuth: raw.azimuth ?? 0,
    timestamp: new Date(raw.timestamp ?? Date.now()),
  };
}

export function normalizeHandover(raw: any): HandoverEvent {
  const status = raw.status ?? (raw.success ? 'success' : 'failed');
  return {
    id: raw.id,
    timestamp: new Date(raw.timestamp),
    sessionId: raw.session_id ?? raw.sessionId,
    fromGateway: raw.from_gateway ?? raw.fromGateway ?? raw.source_gateway_id ?? raw.sourceGatewayId,
    toGateway: raw.to_gateway ?? raw.toGateway ?? raw.target_gateway_id ?? raw.targetGatewayId,
    durationMs: raw.duration_ms ?? raw.durationMs,
    packetLoss: raw.packet_loss ?? raw.packetLoss ?? 0,
    success: status === 'success',
  };
}

export function normalizeDevice(raw: any): Device {
  return {
    deviceId: raw.device_id ?? raw.deviceId,
    mac: raw.mac,
    hardwareId: raw.hw_id ?? raw.hardwareId,
    model: raw.model || undefined,
    owner: raw.owner || undefined,
    status: raw.status,
    provisioningToken: raw.provisioning_token || undefined,
    registeredAt: new Date(raw.registered_at ?? raw.registeredAt ?? Date.now()),
    revokedAt: raw.revoked_at ? new Date(raw.revoked_at) : undefined,
  };
}

function normalizeAlert(raw: any): SecurityAlert {
  return {
    id: raw.id ?? crypto.randomUUID(),
    type: raw.type ?? 'anomaly',
    severity: raw.severity ?? 'medium',
    message: raw.message ?? 'Security anomaly detected',
    timestamp: new Date(raw.timestamp ?? Date.now()),
    mac: raw.mac ?? raw.router_mac,
    deviceId: raw.device_id ?? raw.deviceId,
  };
}

export async function getGateways(): Promise<Gateway[]> {
  const body = await requestJson<{ gateways?: any[]; data?: any[] }>('/gateways');
  return (body.gateways ?? body.data ?? []).map(normalizeGateway);
}

export async function getSessions(): Promise<Session[]> {
  const body = await requestJson<{ sessions?: any[]; data?: any[] }>('/sessions');
  return (body.sessions ?? body.data ?? []).map(normalizeSession);
}

export async function getSatellites(): Promise<Satellite[]> {
  const res = await fetch(resolveApiUrl('/satellites'));
  if (res.status === 404) return [];
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  const body = await res.json();
  return (body.satellites ?? []).map(normalizeSatellite);
}

export async function getHandovers(): Promise<HandoverEvent[]> {
  const body = await requestJson<{ handovers?: any[] }>('/handovers');
  return (body.handovers ?? []).map(normalizeHandover);
}

export async function triggerHandover(sessionId: string, targetGatewayId: string): Promise<HandoverTriggerResponse> {
  const body = await requestJson<any>('/handover/trigger', {
    method: 'POST',
    body: JSON.stringify({ session_id: sessionId, target_gateway_id: targetGatewayId }),
  });
  return {
    handoverId: body.handover_id ?? body.handoverId,
    etaMs: body.eta_ms ?? body.etaMs,
    handover: body.handover ? normalizeHandover(body.handover) : undefined,
  };
}

export async function getDevices(): Promise<Device[]> {
  if (!DEVICES_ENABLED) return [];
  const res = await fetch(resolveApiUrl('/devices'), {
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (res.status === 404) {
    return [];
  }
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`;
    try {
      const body = await res.json();
      message = body.error ?? message;
    } catch {
      // Keep HTTP status as the fallback error.
    }
    throw new Error(message);
  }

  const body = await res.json();
  return (body.devices ?? []).map(normalizeDevice);
}

export async function suspendDevice(deviceId: string): Promise<void> {
  if (!DEVICES_ENABLED) return;
  await requestJson(`/devices/suspend`, {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId }), // Sửa lại thành body
  });
}

export async function revokeDevice(deviceId: string, reason = 'Revoked by admin'): Promise<void> {
  if (!DEVICES_ENABLED) return;
  await requestJson(`/devices/revoke`, {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId, reason }), // Sửa lại thành body
  });
}

export async function activateDevice(deviceId: string): Promise<void> {
  if (!DEVICES_ENABLED) return;
  // POST /api/devices/activate — admin-only reinstatement of a suspended device.
  // Requires only device_id; no CSR/signature ceremony needed for admin-initiated activation.
  await requestJson(`/devices/activate`, {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId }),
  });
}

export function telemetrySocketUrl(token: string | null, topics = 'gateways,sessions,handovers,satellites,telemetry'): string {
  const params = new URLSearchParams({ topics });
  if (token) params.set('token', token);
  const httpUrl = resolveApiUrl('/telemetry/stream');
  const url = new URL(httpUrl, typeof window === 'undefined' ? 'http://localhost' : window.location.origin);
  url.search = params.toString();
  url.protocol = url.protocol.replace('http', 'ws');
  return url.toString();
}

export function parseSecurityAlerts(payload: any): SecurityAlert[] {
  const alerts = payload?.alerts ?? payload?.security_alerts ?? payload?.securityAlerts ?? [];
  if (!Array.isArray(alerts)) return [];
  return alerts.map(normalizeAlert);
}
