/**
 * Forest API
 *
 * Forest Map (network topology visualization) API endpoints
 */

import type { ApiClient } from '../apiClient';
import type { SessionsApi } from './sessions';
import type { GraphData, Node as GraphNode, Edge as GraphEdge } from '../../types/graph';

interface BackendForestResponse {
  nodes: Array<{
    id: string;
    type: string;
    label: string;
    status: string;
    properties: Record<string, unknown>;
    createdAt: string;
    updatedAt: string;
  }>;
  edges: Array<{
    id: string;
    source: string;
    target: string;
    type: string;
    label: string;
    properties: Record<string, unknown>;
    createdAt: string;
  }>;
}

export class ForestApi {
  constructor(
    private client: ApiClient,
    private sessionsApi: SessionsApi
  ) {}

  /**
   * Get graph data from /api/forest endpoint
   */
  async getGraphData(): Promise<GraphData> {
    const response = await this.client.get<BackendForestResponse>('/api/forest');

    const nodes: GraphNode[] = response.nodes.map(node => ({
      id: node.id,
      type: node.type as GraphNode['type'],
      label: node.label,
      status: node.status as GraphNode['status'],
      properties: node.properties,
      createdAt: new Date(node.createdAt),
      updatedAt: new Date(node.updatedAt),
    }));

    const edges: GraphEdge[] = response.edges.map(edge => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      type: edge.type as GraphEdge['type'],
      label: edge.label,
      properties: edge.properties,
      createdAt: new Date(edge.createdAt),
    }));

    return { nodes, edges };
  }

  /**
   * Refresh graph data
   * Get session information again and rebuild the graph
   */
  async refreshGraphData(): Promise<GraphData> {
    return this.getGraphData();
  }

  /**
   * Execute AI analysis
   * POST /api/forest/analyze (not implemented)
   *
 * Future implementation with Llama integration
   */
  async analyzeGraph(_options?: {
    focus?: 'attack_paths' | 'critical_nodes' | 'recommendations';
    depth?: number;
  }): Promise<unknown> {
    // Currently not implemented
    throw new Error('Graph analysis is not yet implemented on the server');

    // Future implementation:
    // return this.client.post('/api/forest/analyze', _options);
  }

  /**
   * Get node details
   * Get node information from session ID
   */
  async getNodeDetails(nodeId: string) {
    // C2 server node case
    if (nodeId === 'c2-server') {
      return {
        id: 'c2-server',
        type: 'system',
        label: 'C2 Server',
        status: 'active',
      };
    }

    // Session node case
    const session = await this.sessionsApi.getSession(nodeId);
    if (!session) {
      throw new Error(`Node ${nodeId} not found`);
    }

    return {
      id: session.id,
      type: 'workstation',
      label: session.hostname,
      status: session.status,
      properties: {
        hostname: session.hostname,
        ip: session.ip,
        os: session.os,
        username: session.username,
        privilege: session.privilege,
      },
    };
  }
}
