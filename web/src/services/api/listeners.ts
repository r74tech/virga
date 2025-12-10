/**
 * Listeners API
 *
 * API endpoints for listener management
 */

import type { ApiClient } from '../apiClient';
import type {
  BackendListener,
  BackendListenersResponse,
  BackendCreateListenerResponse,
  BackendDeleteListenerResponse,
} from '../../types/backend';

export interface ListenerConfig {
  name: string;
  type: string;
  bind_address: string;
  port: number;
  use_ssl?: boolean;
  uri_path: string;
  ssl?: {
    cert: string;
    key: string;
  };
  encryption: {
    type: string;
    key: string;
  };
}

export class ListenersApi {
  constructor(private client: ApiClient) {}

  /**
   * Get listener list
   * GET /api/listeners
   */
  async getListeners(): Promise<BackendListener[]> {
    const response = await this.client.get<BackendListenersResponse>('/api/listeners');
    return response.listeners;
  }

  /**
   * Create listener
   * POST /api/listeners
   */
  async createListener(config: ListenerConfig): Promise<BackendListener> {
    const response = await this.client.post<BackendCreateListenerResponse>(
      '/api/listeners',
      config
    );
    return response.listener;
  }

  /**
   * Delete listener
   * DELETE /api/listeners/{name}
   */
  async deleteListener(name: string): Promise<void> {
    await this.client.delete<BackendDeleteListenerResponse>(
      `/api/listeners/${name}`
    );
  }

  /**
   * Get listener details
   * Get information about a specific listener
   */
  async getListener(name: string): Promise<BackendListener | null> {
    const listeners = await this.getListeners();
    return listeners.find((l) => l.name === name) || null;
  }
}
