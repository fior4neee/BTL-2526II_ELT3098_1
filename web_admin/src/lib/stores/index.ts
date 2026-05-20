import { get, writable } from 'svelte/store';
import type {
  DashboardMetrics,
  Device,
  Gateway,
  HandoverEvent,
  SecurityAlert,
  Satellite,
  Session,
  TelemetrySeries,
  TelemetrySeriesPoint,
} from '$lib/types';
import * as api from '$lib/api/client';

export const gateways = writable<Gateway[]>([]);
export const sessions = writable<Session[]>([]);
export const satellites = writable<Satellite[]>([]);
export const handovers = writable<HandoverEvent[]>([]);
export const devices = writable<Device[]>([]);
export const securityAlerts = writable<SecurityAlert[]>([]);
export const selectedGateway = writable<string | null>(null);
export const lastUpdated = writable<Date | null>(null);
export const lastError = writable<string | null>(null);
export const telemetryConnected = writable(false);
export const telemetryMode = writable<'STREAM' | 'POLLING'>('POLLING');
export const metrics = writable<DashboardMetrics>({
  gatewaysOnline: 0,
  gatewaysOffline: 0,
  activeSessions: 0,
  activeSatellites: 0,
  totalTrafficGbps: 0,
  avgHandoverMs: 0,
  packetLossPct: 0,
  handoverRatePerMin: 0,
  gatewayUptimePct: 0,
});
export const series = writable<TelemetrySeries>({
  trafficGbps: [],
  handoverCount: [],
  sessionCount: [],
});

let refreshInterval: ReturnType<typeof setInterval> | null = null;
let telemetrySocket: WebSocket | null = null;

const MAX_SERIES_POINTS = 48;

export async function refreshAll() {
  try {
    const [gatewayData, sessionData, satelliteData, handoverData] = await Promise.all([
      api.getGateways(),
      api.getSessions(),
      api.getSatellites(),
      api.getHandovers(),
    ]);

    gateways.set(gatewayData);
    sessions.set(sessionData);
    satellites.set(satelliteData);
    handovers.set(handoverData);
    lastUpdated.set(new Date());
    lastError.set(null);

    recomputeMetrics();
    pushSeriesPoints(sessionData, handoverData);
  } catch (err) {
    lastError.set(err instanceof Error ? err.message : 'Failed to refresh API data');
  }
}

export async function refreshDevices() {
  try {
    devices.set(await api.getDevices());
    lastUpdated.set(new Date());
    lastError.set(null);
  } catch (err) {
    lastError.set(err instanceof Error ? err.message : 'Failed to refresh devices');
  }
}

export async function triggerHandoverForSession(sessionId: string, targetGatewayId: string) {
  const result = await api.triggerHandover(sessionId, targetGatewayId);
  await refreshAll();
  return result;
}

export async function suspendDeviceByMac(mac: string) {
  await api.suspendDevice(mac);
  await refreshDevices();
}

export async function revokeDeviceByMac(mac: string) {
  await api.revokeDeviceByMac(mac);
  await refreshDevices();
}

export function startLiveUpdates(intervalMs = 5000) {
  void refreshAll();

  if (!refreshInterval) {
    refreshInterval = setInterval(() => void refreshAll(), intervalMs);
  }

  startTelemetrySocket();
}

export function stopLiveUpdates() {
  if (refreshInterval) {
    clearInterval(refreshInterval);
    refreshInterval = null;
  }

  if (telemetrySocket) {
    telemetrySocket.close();
    telemetrySocket = null;
  }
  telemetryConnected.set(false);
  telemetryMode.set('POLLING');
}

function startTelemetrySocket() {
  if (telemetrySocket || typeof WebSocket === 'undefined') return;

  const token = import.meta.env.VITE_TELEMETRY_TOKEN ?? null;
  if (!token) {
    telemetryMode.set('POLLING');
    return;
  }
  const socketUrl = api.telemetrySocketUrl(token);
  telemetrySocket = new WebSocket(socketUrl);

  telemetrySocket.onopen = () => {
    telemetryConnected.set(true);
    telemetryMode.set('STREAM');
  };
  telemetrySocket.onerror = () => {
    telemetryConnected.set(false);
    telemetryMode.set('POLLING');
  };
  telemetrySocket.onclose = () => {
    telemetryConnected.set(false);
    telemetryMode.set('POLLING');
  };
  telemetrySocket.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data);
      applyTelemetryPayload(payload);
    } catch {
      // Ignore invalid payloads
    }
  };
}

function applyTelemetryPayload(payload: any) {
  let updated = false;

  if (Array.isArray(payload?.gateways)) {
    gateways.set(payload.gateways.map((raw: unknown) => api.normalizeGateway(raw)));
    updated = true;
  }
  if (Array.isArray(payload?.sessions)) {
    sessions.set(payload.sessions.map((raw: unknown) => api.normalizeSession(raw)));
    updated = true;
  }
  if (Array.isArray(payload?.handovers)) {
    handovers.set(payload.handovers.map((raw: unknown) => api.normalizeHandover(raw)));
    updated = true;
  }
  if (Array.isArray(payload?.satellites)) {
    satellites.set(payload.satellites.map((raw: unknown) => api.normalizeSatellite(raw)));
    updated = true;
  }
  if (Array.isArray(payload?.devices)) {
    devices.set(payload.devices.map((raw: unknown) => api.normalizeDevice(raw)));
    updated = true;
  }

  const alerts = api.parseSecurityAlerts(payload);
  if (alerts.length) {
    securityAlerts.update((current: SecurityAlert[]) => {
      const merged = [...alerts, ...current].slice(0, 100);
      return merged;
    });
  }

  if (updated) {
    lastUpdated.set(new Date());
    recomputeMetrics();
    pushSeriesPoints(get(sessions), get(handovers));
  }
}

function recomputeMetrics() {
  const gatewayList = get(gateways);
  const sessionList = get(sessions);
  const handoverList = get(handovers);
  const satelliteList = get(satellites);

  const online = gatewayList.filter((g: Gateway) => g.status === 'alive').length;
  const offline = gatewayList.length - online;
  const avgHandoverMs = handoverList.length
    ? handoverList.reduce((acc: number, h: HandoverEvent) => acc + h.durationMs, 0) / handoverList.length
    : 0;
  const packetLossPct = handoverList.length
    ? (handoverList.reduce((acc: number, h: HandoverEvent) => acc + (h.packetLoss ?? 0), 0) / handoverList.length) * 100
    : 0;
  const totalTrafficGbps = sessionList.reduce((acc: number, s: Session) => acc + (s.dataMb ?? 0), 0) / 1024;
  const handoverRatePerMin = handoverList.filter((h: HandoverEvent) => Date.now() - h.timestamp.getTime() < 60_000).length;
  const gatewayUptimePct = gatewayList.length
    ? gatewayList.reduce((acc: number, g: Gateway) => acc + (g.uptimePct ?? 99), 0) / gatewayList.length
    : 0;

  metrics.set({
    gatewaysOnline: online,
    gatewaysOffline: offline,
    activeSessions: sessionList.length,
    activeSatellites: satelliteList.length,
    totalTrafficGbps: Number(totalTrafficGbps.toFixed(2)),
    avgHandoverMs: Number(avgHandoverMs.toFixed(1)),
    packetLossPct: Number(packetLossPct.toFixed(3)),
    handoverRatePerMin,
    gatewayUptimePct: Number(gatewayUptimePct.toFixed(2)),
  });
}

function pushSeriesPoints(sessionList: Session[], handoverList: HandoverEvent[]) {
  const now = new Date();
  const trafficGbps = sessionList.reduce((acc, s) => acc + (s.dataMb ?? 0), 0) / 1024;
  const handoverCount = handoverList.filter((h) => now.getTime() - h.timestamp.getTime() < 60_000).length;
  const sessionCount = sessionList.length;

  series.update((current: TelemetrySeries) => ({
    trafficGbps: appendPoint(current.trafficGbps, { ts: now, value: Number(trafficGbps.toFixed(2)) }),
    handoverCount: appendPoint(current.handoverCount, { ts: now, value: handoverCount }),
    sessionCount: appendPoint(current.sessionCount, { ts: now, value: sessionCount }),
  }));
}

function appendPoint(list: TelemetrySeriesPoint[], point: TelemetrySeriesPoint) {
  const next = [...list, point];
  return next.slice(Math.max(0, next.length - MAX_SERIES_POINTS));
}
