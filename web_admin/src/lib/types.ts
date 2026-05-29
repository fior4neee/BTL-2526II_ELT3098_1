export type GatewayStatus = 'alive' | 'degraded' | 'dead';
export type SessionState = 'Acquire' | 'Hold' | 'Prepare' | 'Execute' | 'Release';
export type DeviceStatus = 'registered' | 'active' | 'suspended' | 'revoked';
export type AlertSeverity = 'low' | 'medium' | 'high';

export interface Gateway {
  id: string;
  name: string;
  lat: number;
  lng: number;
  altitudeKm: number;
  status: GatewayStatus;
  currentSessions: number;
  maxSessions: number;
  minElevationDeg: number;
  antennaGainDbi: number;
  beamWidthDeg: number;
  cpuPct?: number;
  memoryPct?: number;
  bandwidthMbps?: number;
  uptimePct?: number;
  lastSeen?: Date;
}

export interface Session {
  id: string;
  routerMac: string;
  satelliteId: string;
  currentGatewayId: string;
  nextGatewayId?: string;
  state: SessionState;
  startTime: Date;
  dataMb?: number;
  cnRatioDb?: number;
  lastActivity?: Date;
}

export interface Satellite {
  satelliteId: string;
  latitude: number;
  longitude: number;
  altitude: number;
  elevation: number;
  azimuth: number;
  timestamp: Date;
}

export interface HandoverEvent {
  id: string;
  timestamp: Date;
  sessionId: string;
  fromGateway: string;
  toGateway: string;
  durationMs: number;
  packetLoss: number;
  success: boolean;
}

export interface Device {
  deviceId: string;
  mac: string;
  hardwareId: string;
  model?: string;
  owner?: string;
  status: DeviceStatus;
  provisioningToken?: string;
  registeredAt: Date;
  revokedAt?: Date;
}

export interface HandoverTriggerResponse {
  handoverId: string;
  etaMs: number;
  handover?: HandoverEvent;
}

export interface SecurityAlert {
  id: string;
  type: 'unregistered' | 'mac_change' | 'geofence' | 'anomaly';
  severity: AlertSeverity;
  message: string;
  timestamp: Date;
  mac?: string;
  deviceId?: string;
}

export interface DashboardMetrics {
  gatewaysOnline: number;
  gatewaysOffline: number;
  activeSessions: number;
  activeSatellites: number;
  totalTrafficGbps: number;
  avgHandoverMs: number;
  packetLossPct: number;
  handoverRatePerMin: number;
  gatewayUptimePct: number;
}

export interface TelemetrySeriesPoint {
  ts: Date;
  value: number;
}

export interface TelemetrySeries {
  trafficGbps: TelemetrySeriesPoint[];
  handoverCount: TelemetrySeriesPoint[];
  sessionCount: TelemetrySeriesPoint[];
}
