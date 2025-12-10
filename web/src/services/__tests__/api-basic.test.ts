/**
 * API Basic Tests
 *
 * Basic API operation tests
 * These tests require the actual server to be running
 */

import { describe, it, expect } from 'vitest';
import { api } from '../api';
import { config } from '../../config/env';

describe('API Configuration', () => {
  it('should load environment config', () => {
    expect(config.apiBaseURL).toBeDefined();
    expect(config.apiTimeout).toBeGreaterThan(0);
  });

  it('should have api instance', () => {
    expect(api).toBeDefined();
    expect(api.sessions).toBeDefined();
    expect(api.listeners).toBeDefined();
    expect(api.extensions).toBeDefined();
    expect(api.forest).toBeDefined();
  });
});

describe('Type Converters', () => {
  it('should calculate session status correctly', async () => {
    const {
      calculateSessionStatus,
      extractIPFromRemoteAddr,
    } = await import('../converters');

    // Current time should be active
    const now = new Date().toISOString();
    expect(calculateSessionStatus(now)).toBe('active');

    // 2 minutes ago should be inactive
    const twoMinutesAgo = new Date(Date.now() - 2 * 60 * 1000).toISOString();
    expect(calculateSessionStatus(twoMinutesAgo)).toBe('inactive');

    // 10 minutes ago should be disconnected
    const tenMinutesAgo = new Date(Date.now() - 10 * 60 * 1000).toISOString();
    expect(calculateSessionStatus(tenMinutesAgo)).toBe('disconnected');
  });

  it('should extract IP from remote_addr', async () => {
    const { extractIPFromRemoteAddr } = await import('../converters');

    expect(extractIPFromRemoteAddr('192.168.1.10:12345')).toBe('192.168.1.10');
    expect(extractIPFromRemoteAddr('10.0.0.1:8080')).toBe('10.0.0.1');
  });
});

// These tests require the actual server to be running
describe.skip('Sessions API (requires server)', () => {
  it('should fetch sessions', async () => {
    const sessions = await api.sessions.getSessions();
    expect(Array.isArray(sessions)).toBe(true);
  });

  it('should have correct session structure', async () => {
    const sessions = await api.sessions.getSessions();
    if (sessions.length > 0) {
      const session = sessions[0];
      expect(session).toHaveProperty('id');
      expect(session).toHaveProperty('hostname');
      expect(session).toHaveProperty('username');
      expect(session).toHaveProperty('os');
      expect(session).toHaveProperty('status');
    }
  });
});

describe.skip('Listeners API (requires server)', () => {
  it('should fetch listeners', async () => {
    const listeners = await api.listeners.getListeners();
    expect(Array.isArray(listeners)).toBe(true);
  });
});

describe.skip('Forest API (requires server)', () => {
  it('should build graph from sessions', async () => {
    const graphData = await api.forest.getGraphData();
    expect(graphData).toHaveProperty('nodes');
    expect(graphData).toHaveProperty('edges');
    expect(Array.isArray(graphData.nodes)).toBe(true);
    expect(Array.isArray(graphData.edges)).toBe(true);

    // Verify C2 server node exists
    const c2Node = graphData.nodes.find((n) => n.id === 'c2-server');
    expect(c2Node).toBeDefined();
    expect(c2Node?.type).toBe('system');
  });
});
