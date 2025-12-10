/**
 * Type Converters
 *
 * Backend and frontend type conversions
 */

import type { BackendSession, BackendTaskResult, BackendListener } from '../types/backend';
import type { Session, SessionStatus } from '../types/session';
import type { Node, NodeStatus, Edge } from '../types/graph';
import type { Listener, ListenerStatus, ListenerType } from '../types/listener';

/**
 * Calculate session status based on last activity
 * Determine status based on elapsed time since last activity
 */
export function calculateSessionStatus(lastActivity: string): SessionStatus {
  const now = Date.now();
  const lastActivityTime = new Date(lastActivity).getTime();
  const diffSeconds = (now - lastActivityTime) / 1000;

  if (diffSeconds < 60) return 'active'; // Within 1 minute
  if (diffSeconds < 300) return 'inactive'; // Within 5 minutes
  return 'disconnected'; // More than 5 minutes
}

/**
 * Extract IP address from remote_addr
 * "192.168.1.10:12345" -> "192.168.1.10"
 */
export function extractIPFromRemoteAddr(remoteAddr: string): string {
  const parts = remoteAddr.split(':');
  return parts[0] || remoteAddr;
}

/**
 * Backend session -> frontend session
 */
export function convertBackendSession(backend: BackendSession): Session {
  return {
    id: backend.id,
    agentId: backend.id,
    hostname: backend.hostname,
    username: backend.username,
    os: backend.os,
    arch: backend.environment.arch,
    status: calculateSessionStatus(backend.last_activity),
    connectionType: 'direct', // TODO: Detect pivot connections
    lastActivity: new Date(backend.last_activity),
    ip: extractIPFromRemoteAddr(backend.remote_addr),
    processId: backend.environment.process_id,
    interval: backend.interval ?? 30,
    jitter: backend.jitter ?? 10,
    metadata: {
      lastCheckin: new Date(backend.last_activity),
      beaconInterval: backend.interval ?? 30,
      jitter: backend.jitter ?? 10,
    },
  };
}

/**
 * Convert session to Forest Map node
 */
export function convertSessionToNode(session: Session): Node {
  // Map session status to node status
  const statusMap: Record<SessionStatus, NodeStatus> = {
    active: 'active',
    inactive: 'inactive',
    disconnected: 'disconnected',
  };

  // Determine node type from OS
  const getNodeType = (os: string): 'workstation' | 'server' => {
    const osLower = os.toLowerCase();
    if (osLower.includes('server')) return 'server';
    return 'workstation';
  };

  return {
    id: session.id,
    type: getNodeType(session.os),
    label: session.hostname,
    status: statusMap[session.status],
    properties: {
      hostname: session.hostname,
      ip: session.ip,
      os: session.os,
      username: session.username,
      privilege: session.privilege,
    },
    createdAt: session.lastActivity,
    updatedAt: session.lastActivity,
  };
}

/**
 * Create C2 server node
 */
export function createC2ServerNode(): Node {
  return {
    id: 'c2-server',
    type: 'system',
    label: 'C2 Server',
    status: 'active',
    properties: {
      hostname: 'virga-c2',
    },
    createdAt: new Date(),
    updatedAt: new Date(),
  };
}

/**
 * Create edge between C2 server and session
 */
export function createC2Edge(sessionId: string): Edge {
  return {
    id: `edge-c2-${sessionId}`,
    source: 'c2-server',
    target: sessionId,
    type: 'c2_connection',
    label: 'C2',
    weight: 1.0,
    properties: {},
    createdAt: new Date(),
    metadata: {
      protocol: 'https',
      port: 8080,
    },
  };
}

/**
 * Build graph data from session list
 */
export function buildGraphFromSessions(sessions: Session[]): {
  nodes: Node[];
  edges: Edge[];
} {
  const nodes: Node[] = [
    createC2ServerNode(),
    ...sessions.map(convertSessionToNode),
  ];

  const edges: Edge[] = sessions.map((session) => createC2Edge(session.id));

  return { nodes, edges };
}

/**
 * Get task type display name
 */
export function getTaskTypeDisplayName(type: string): string {
  const typeMap: Record<string, string> = {
    shell: 'Shell Command',
    upload: 'File Upload',
    download: 'File Download',
    cd: 'Change Directory',
    pwd: 'Print Working Directory',
    ps: 'Process List',
    kill: 'Kill Process',
    netstat: 'Network Connections',
    portfwd: 'Port Forward',
    exec: 'Execute Binary',
    sleep: 'Sleep Interval',
    jitter: 'Jitter Setting',
  };

  return typeMap[type] || type;
}

/**
 * Convert task result to terminal output format
 */
export function formatTaskResultForTerminal(result: BackendTaskResult): string {
  if (result.error) {
    return `Error: ${result.error}`;
  }

  if (result.exit_code !== 0) {
    return `Command failed with exit code ${result.exit_code}\n${result.output}`;
  }

  return result.output;
}

/**
 * Get icon name from OS information (for future implementation)
 */
export function getOSIcon(os: string): string {
  const osLower = os.toLowerCase();
  if (osLower.includes('windows')) return 'windows';
  if (osLower.includes('linux')) return 'linux';
  if (osLower.includes('darwin') || osLower.includes('macos')) return 'apple';
  return 'computer';
}

/**
 * Determine privilege level
 * Will be updated when backend provides this information
 */
export function determinePrivilegeLevel(username: string, os: string): string {
  const usernameLower = username.toLowerCase();
  const osLower = os.toLowerCase();

  if (osLower.includes('windows')) {
    if (
      usernameLower.includes('admin') ||
      usernameLower.includes('system') ||
      usernameLower === 'nt authority\\system'
    ) {
      return 'admin';
    }
  } else {
    if (usernameLower === 'root') return 'admin';
  }

  return 'user';
}

/**
 * Convert backend listener to frontend type
 */
export function convertBackendListener(backend: BackendListener): Listener {
  return {
    name: backend.name,
    type: backend.type as ListenerType,
    bindAddress: backend.bind_address,
    port: backend.port,
    status: backend.status as ListenerStatus,
    metadata: {
      tlsEnabled: backend.config?.tls_enabled as boolean | undefined,
      certFile: backend.config?.cert_file as string | undefined,
      keyFile: backend.config?.key_file as string | undefined,
    },
  };
}

/**
 * Convert frontend listener to backend type
 */
export function convertListenerToBackend(listener: Listener): BackendListener {
  return {
    name: listener.name,
    type: listener.type,
    bind_address: listener.bindAddress,
    port: listener.port,
    status: listener.status,
    config: {
      tls_enabled: listener.metadata?.tlsEnabled,
      cert_file: listener.metadata?.certFile,
      key_file: listener.metadata?.keyFile,
    },
  };
}
