import type {
  Device,
  Gateway,
  HandoverEvent,
  HandoverTriggerResponse,
  Satellite,
  Session,
} from '$lib/types';

const API_BASE = (import.meta.env.VITE_CORE_API_BASE ?? '/api/v1').replace(/\/$/, '');

async function requestJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
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

function normalizeGateway(raw: any): Gateway {
  return {
    id: raw.id,
    name: raw.name,
    lat: raw.location?.lat ?? 0,
    lng: raw.location?.lon ?? 0,
    altitudeKm: raw.location?.alt ?? 0,
    status: gatewayStatus(raw.status),
    currentSessions: raw.current_sessions ?? 0,
    maxSessions: raw.max_sessions ?? 0,
    minElevationDeg: raw.min_elevation_deg ?? 0,
    antennaGainDbi: raw.antenna?.gain_dbi ?? 0,
    beamWidthDeg: raw.antenna?.beam_width_deg ?? 0,
  };
}

function normalizeSession(raw: any): Session {
  return {
    id: raw.session_id,
    routerMac: raw.router_mac,
    satelliteId: raw.satellite_id,
    currentGatewayId: raw.current_gateway_id,
    nextGatewayId: raw.next_gateway_id || undefined,
    state: raw.state,
    startTime: new Date(raw.start_time),
  };
}

function normalizeSatellite(raw: any): Satellite {
  return {
    satelliteId: raw.satellite_id,
    latitude: raw.latitude,
    longitude: raw.longitude,
    altitude: raw.altitude,
    elevation: raw.elevation,
    azimuth: raw.azimuth,
    timestamp: new Date(raw.timestamp),
  };
}

function normalizeHandover(raw: any): HandoverEvent {
  return {
    id: raw.id,
    timestamp: new Date(raw.timestamp),
    sessionId: raw.session_id,
    fromGateway: raw.from_gateway,
    toGateway: raw.to_gateway,
    durationMs: raw.duration_ms,
    packetLoss: raw.packet_loss,
    success: raw.success,
  };
}

function normalizeDevice(raw: any): Device {
  return {
    deviceId: raw.device_id,
    mac: raw.mac,
    hardwareId: raw.hw_id,
    model: raw.model || undefined,
    status: raw.status,
    provisioningToken: raw.provisioning_token || undefined,
    registeredAt: new Date(raw.registered_at),
    revokedAt: raw.revoked_at ? new Date(raw.revoked_at) : undefined,
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
  const res = await fetch(`${API_BASE}/satellites`);
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
    handoverId: body.handover_id,
    etaMs: body.eta_ms,
    handover: normalizeHandover(body.handover),
  };
}

export async function getDevices(): Promise<Device[]> {
  const res = await fetch(`${API_BASE}/devices`, {
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

export async function suspendDevice(mac: string): Promise<void> {
  await requestJson(`/devices/${encodeURIComponent(mac)}/suspend`, { method: 'POST' });
}

export async function revokeDevice(deviceId: string, reason = 'Revoked by admin'): Promise<void> {
  await requestJson('/devices/revoke', {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId, reason }),
  });
}

export function telemetryStreamUrl(topics = 'gateways,sessions,handovers'): string {
  const params = new URLSearchParams({ topics });
  return `${API_BASE}/telemetry/stream?${params.toString()}`;
}
