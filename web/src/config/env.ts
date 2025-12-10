/**
 * Environment Configuration
 *
 * Load Vite environment variables and provide configuration for the application
 */

export const config = {
  // API Base URL
  apiBaseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8443',

  // WebSocket URL
  wsURL: import.meta.env.VITE_WS_URL || 'ws://localhost:8443/api/ws',

  // Debug mode
  debugMode: import.meta.env.VITE_DEBUG_MODE === 'true',

  // API timeout (milliseconds)
  apiTimeout: Number(import.meta.env.VITE_API_TIMEOUT) || 10000,

  // Polling interval (milliseconds)
  pollingInterval: Number(import.meta.env.VITE_POLLING_INTERVAL) || 5000,
} as const;

// Type-safe configuration object
export type Config = typeof config;

// Debug log output
if (config.debugMode) {
  console.log('[Config] Environment loaded:', {
    apiBaseURL: config.apiBaseURL,
    wsURL: config.wsURL,
    debugMode: config.debugMode,
  });
}
