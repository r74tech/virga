export type NodeType = 'workstation' | 'server' | 'system' | 'network';
export type NodeStatus = 'active' | 'inactive' | 'disconnected';
export type EdgeType = 'c2_connection' | 'lateral_movement' | 'data_flow';

export interface Node {
  id: string;
  type: NodeType;
  label: string;
  status: NodeStatus;
  properties: NodeProperties;
  createdAt: Date;
  updatedAt: Date;
}

export interface NodeProperties {
  hostname?: string;
  ip?: string;
  os?: string;
  username?: string;
  privilege?: string;
  [key: string]: any;
}

export interface Edge {
  id: string;
  source: string;
  target: string;
  type: EdgeType;
  label?: string;
  weight?: number;
  properties?: EdgeProperties;
  createdAt: Date;
  metadata?: EdgeMetadata;
}

export interface EdgeProperties {
  bandwidth?: string;
  latency?: number;
  [key: string]: any;
}

export interface EdgeMetadata {
  protocol?: string;
  port?: number;
  established?: Date;
  [key: string]: any;
}

export interface GraphData {
  nodes: Node[];
  edges: Edge[];
}

export interface AnalysisResult {
  criticalNodes: NodeAnalysis[];
  attackPaths: AttackPath[];
  recommendations: Recommendation[];
}

export interface NodeAnalysis {
  nodeId: string;
  importance: number;
  reason: string;
  connections: number;
}

export interface AttackPath {
  path: string[];
  difficulty: 'easy' | 'medium' | 'hard';
  impact: 'low' | 'medium' | 'high';
  description: string;
}

export interface Recommendation {
  target: string;
  action: string;
  reason: string;
  priority: number;
}
