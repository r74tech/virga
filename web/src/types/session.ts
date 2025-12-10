export interface Session {
  id: string;
  agentId: string;
  hostname: string;
  username: string;
  os: string;
  arch: string;
  status: SessionStatus;
  connectionType: ConnectionType;
  pivotChain?: string[];
  lastActivity: Date;
  ip: string;
  privilege?: string;
  processId?: number;
  interval?: number;
  jitter?: number;
  metadata?: SessionMetadata;
}

export type SessionStatus = 'active' | 'inactive' | 'disconnected';
export type ConnectionType = 'direct' | 'pivot';

export interface SessionMetadata {
  beaconInterval?: number;
  jitter?: number;
  lastCheckin?: Date;
  missedCheckins?: number;
  [key: string]: any;
}

export interface SessionStats {
  totalSessions: number;
  activeSessions: number;
  inactiveSessions: number;
  disconnectedSessions: number;
}
