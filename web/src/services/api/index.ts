/**
 * API Index
 *
 * Entry point for all API endpoints
 */

import { apiClient } from '../apiClient';
import { SessionsApi } from './sessions';
import { ListenersApi } from './listeners';
import { ExtensionsApi } from './extensions';
import { ForestApi } from './forest';

/**
 * API instance
 * Unified API client for use throughout the application
 */
export const api = {
  sessions: new SessionsApi(apiClient),
  listeners: new ListenersApi(apiClient),
  extensions: new ExtensionsApi(apiClient),
  forest: new ForestApi(apiClient, new SessionsApi(apiClient)),
};

/**
 * Type exports
 */
export type { SessionsApi } from './sessions';
export type { ListenersApi, ListenerConfig } from './listeners';
export type { ExtensionsApi } from './extensions';
export type { ForestApi } from './forest';

/**
 * Example usage:
 *
 * // Get session list
 * const sessions = await api.sessions.getSessions();
 *
 * // Add task
 * const taskId = await api.sessions.addTask('session-id', 'shell', {
 *   command: 'whoami'
 * });
 *
 * // Get listener list
 * const listeners = await api.listeners.getListeners();
 *
 * // Get graph data
 * const graphData = await api.forest.getGraphData();
 *
 * // Upload extension
 * await api.extensions.uploadExtension('session-id', file);
 */
