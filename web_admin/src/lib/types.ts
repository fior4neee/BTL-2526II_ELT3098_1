export type GatewayStatus = 'alive' | 'degraded' | 'dead';
export type SessionState = 'Acquire' | 'Hold' | 'Prepare' | 'Execute' | 'Release';
export type DeviceStatus = 'registered' | 'active' | 'suspended' | 'revoked';

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
}

export interface Session {
  id: string;
  routerMac: string;
  satelliteId: string;
  currentGatewayId: string;
  nextGatewayId?: string;
  state: SessionState;
  startTime: Date;
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
  status: DeviceStatus;
  provisioningToken?: string;
  registeredAt: Date;
  revokedAt?: Date;
}

export interface HandoverTriggerResponse {
  handoverId: string;
  etaMs: number;
  handover: HandoverEvent;
}
