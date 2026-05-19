import { writable } from 'svelte/store';
import type { Device, Gateway, HandoverEvent, Satellite, Session } from '$lib/types';
import * as api from '$lib/api/client';

export const gateways = writable<Gateway[]>([]);
export const sessions = writable<Session[]>([]);
export const satellites = writable<Satellite[]>([]);
export const handovers = writable<HandoverEvent[]>([]);
export const devices = writable<Device[]>([]);
export const selectedGateway = writable<string | null>(null);
export const lastUpdated = writable<Date | null>(null);
export const lastError = writable<string | null>(null);
export const telemetryConnected = writable(false);

let refreshInterval: ReturnType<typeof setInterval> | null = null;
let telemetrySource: EventSource | null = null;

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

export async function revokeDeviceById(deviceId: string) {
  await api.revokeDevice(deviceId);
  await refreshDevices();
}

export function startLiveUpdates(intervalMs = 5000) {
  void refreshAll();

  if (!refreshInterval) {
    refreshInterval = setInterval(() => void refreshAll(), intervalMs);
  }

  startTelemetryStream();
}

export function stopLiveUpdates() {
  if (refreshInterval) {
    clearInterval(refreshInterval);
    refreshInterval = null;
  }

  if (telemetrySource) {
    telemetrySource.close();
    telemetrySource = null;
  }
  telemetryConnected.set(false);
}

function startTelemetryStream() {
  if (telemetrySource || typeof EventSource === 'undefined') return;

  telemetrySource = new EventSource(api.telemetryStreamUrl());
  telemetrySource.onopen = () => telemetryConnected.set(true);
  telemetrySource.onerror = () => telemetryConnected.set(false);
  telemetrySource.addEventListener('telemetry', () => void refreshAll());
}
