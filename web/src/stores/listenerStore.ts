/**
 * Listener Store
 *
 * Listener management state management
 */

import { create } from 'zustand';
import type { Listener, ListenerStats, CreateListenerPayload } from '../types/listener';
import { api } from '../services/api';
import { convertBackendListener } from '../services/converters';

interface ListenerState {
  listeners: Listener[];
  selectedListener: Listener | null;
  stats: ListenerStats;
  isLoading: boolean;
  error: string | null;

  // API methods
  fetchListeners: () => Promise<void>;
  createListener: (payload: CreateListenerPayload) => Promise<void>;
  deleteListener: (name: string) => Promise<void>;
  setSelectedListener: (listener: Listener | null) => void;
  refreshStats: () => void;
  clearError: () => void;
}

const calculateStats = (listeners: Listener[]): ListenerStats => ({
  totalListeners: listeners.length,
  runningListeners: listeners.filter((l) => l.status === 'running').length,
  stoppedListeners: listeners.filter((l) => l.status === 'stopped').length,
  errorListeners: listeners.filter((l) => l.status === 'error').length,
});

export const useListenerStore = create<ListenerState>((set) => ({
  listeners: [],
  selectedListener: null,
  stats: calculateStats([]),
  isLoading: false,
  error: null,

  fetchListeners: async () => {
    set({ isLoading: true, error: null });

    try {
      const backendListeners = await api.listeners.getListeners();
      const listeners = backendListeners.map(convertBackendListener);
      set({
        listeners,
        stats: calculateStats(listeners),
        isLoading: false,
      });
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to fetch listeners';
      set({
        isLoading: false,
        error: errorMessage,
      });
      console.error('Failed to fetch listeners:', error);
    }
  },

  createListener: async (payload: CreateListenerPayload) => {
    set({ isLoading: true, error: null });

    try {
      const backendListener = await api.listeners.createListener({
        name: payload.name,
        type: payload.type,
        bind_address: payload.bindAddress,
        port: payload.port,
        use_ssl: payload.tlsEnabled || false,
        uri_path: payload.uriPath,
        ssl: payload.tlsEnabled && payload.certFile && payload.keyFile ? {
          cert: payload.certFile,
          key: payload.keyFile,
        } : undefined,
        encryption: {
          type: 'aes-256',
          key: payload.encryptionKey,
        },
      });

      const listener = convertBackendListener(backendListener);
      set((state) => {
        const listeners = [...state.listeners, listener];
        return { listeners, stats: calculateStats(listeners), isLoading: false };
      });
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to create listener';
      set({
        isLoading: false,
        error: errorMessage,
      });
      console.error('Failed to create listener:', error);
      throw error;
    }
  },

  deleteListener: async (name: string) => {
    set({ isLoading: true, error: null });

    try {
      await api.listeners.deleteListener(name);
      set((state) => {
        const listeners = state.listeners.filter((l) => l.name !== name);
        const selectedListener =
          state.selectedListener?.name === name ? null : state.selectedListener;
        return {
          listeners,
          selectedListener,
          stats: calculateStats(listeners),
          isLoading: false,
        };
      });
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to delete listener';
      set({
        isLoading: false,
        error: errorMessage,
      });
      console.error('Failed to delete listener:', error);
      throw error;
    }
  },

  setSelectedListener: (listener) => set({ selectedListener: listener }),

  refreshStats: () =>
    set((state) => ({ stats: calculateStats(state.listeners) })),

  clearError: () => set({ error: null }),
}));
