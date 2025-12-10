import { create } from 'zustand';
import type { Session, SessionStats } from '../types/session';
import { api } from '../services/api';

interface SessionState {
  sessions: Session[];
  selectedSession: Session | null;
  stats: SessionStats;
  isLoading: boolean;
  error: string | null;

  // API methods
  fetchSessions: () => Promise<void>;
  setSelectedSession: (session: Session | null) => void;
  updateSession: (sessionId: string, updates: Partial<Session>) => void;
  addSession: (session: Session) => void;
  removeSession: (sessionId: string) => void;
  refreshStats: () => void;
  setInteractiveMode: (sessionId: string, interactive: boolean) => Promise<void>;
  clearError: () => void;
}

const calculateStats = (sessions: Session[]): SessionStats => ({
  totalSessions: sessions.length,
  activeSessions: sessions.filter((s) => s.status === 'active').length,
  inactiveSessions: sessions.filter((s) => s.status === 'inactive').length,
  disconnectedSessions: sessions.filter((s) => s.status === 'disconnected').length,
});

export const useSessionStore = create<SessionState>((set, get) => ({
  sessions: [],
  selectedSession: null,
  stats: calculateStats([]),
  isLoading: false,
  error: null,

  fetchSessions: async () => {
    set({ isLoading: true, error: null });

    try {
      const sessions = await api.sessions.getSessions();
      set({
        sessions,
        stats: calculateStats(sessions),
        isLoading: false,
      });
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to fetch sessions';
      set({
        isLoading: false,
        error: errorMessage,
      });
      console.error('Failed to fetch sessions:', error);
    }
  },

  setSelectedSession: (session) => set({ selectedSession: session }),

  updateSession: (sessionId, updates) =>
    set((state) => {
      const sessions = state.sessions.map((s) =>
        s.id === sessionId ? { ...s, ...updates } : s
      );
      return { sessions, stats: calculateStats(sessions) };
    }),

  addSession: (session) =>
    set((state) => {
      const sessions = [...state.sessions, session];
      return { sessions, stats: calculateStats(sessions) };
    }),

  removeSession: (sessionId) =>
    set((state) => {
      const sessions = state.sessions.filter((s) => s.id !== sessionId);
      // Clear selection if the removed session was selected
      const selectedSession =
        state.selectedSession?.id === sessionId ? null : state.selectedSession;
      return { sessions, selectedSession, stats: calculateStats(sessions) };
    }),

  refreshStats: () =>
    set((state) => ({ stats: calculateStats(state.sessions) })),

  setInteractiveMode: async (sessionId, interactive) => {
    try {
      await api.sessions.setInteractiveMode(sessionId, interactive);
      // Refresh session data on success
      await get().fetchSessions();
    } catch (error) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : 'Failed to set interactive mode';
      set({ error: errorMessage });
      console.error('Failed to set interactive mode:', error);
    }
  },

  clearError: () => set({ error: null }),
}));
