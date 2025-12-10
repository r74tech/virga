/**
 * Terminal Store
 *
 * Manages terminal state for each session, including:
 * - Multiple terminal tabs per session
 * - Command history per terminal
 * - Terminal entries (input/output)
 * - Session-specific state
 * - Persistence to IndexedDB
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { zustandStorage } from '../utils/db';
import { api } from '../services/api';
import { config } from '../config/env';
import { parseCommand } from '../utils/commandParser';
import { executeCommand } from '../services/commandRouter';

export interface TerminalEntry {
  id: string;
  type: 'input' | 'output' | 'error' | 'system';
  content: string;
  timestamp: Date;
}

export interface Terminal {
  id: string;
  sessionId: string;
  name: string;
  entries: TerminalEntry[];
  isExecuting: boolean;
  createdAt: Date;
  isShellMode: boolean;
}

interface TerminalState {
  // Map of terminalId -> terminal state
  terminals: Record<string, Terminal>;

  // Map of sessionId -> array of terminal IDs
  sessionTerminalIds: Record<string, string[]>;

  // Map of sessionId -> active terminal ID
  activeTerminalIds: Record<string, string>;

  // Get active terminal for a session
  getActiveTerminal: (sessionId: string) => Terminal | null;

  // Get all terminals for a session
  getSessionTerminals: (sessionId: string) => Terminal[];

  // Create a new terminal for a session
  createTerminal: (sessionId: string, welcomeMessages?: string[]) => string;

  // Switch to a different terminal
  setActiveTerminal: (sessionId: string, terminalId: string) => void;

  // Close a terminal
  closeTerminal: (terminalId: string) => void;

  // Add entry to terminal
  addEntry: (terminalId: string, entry: Omit<TerminalEntry, 'id' | 'timestamp'>) => void;

  // Set executing state
  setExecuting: (terminalId: string, isExecuting: boolean) => void;

  // Clear terminal
  clearTerminal: (terminalId: string) => void;

  // Execute command (API integration)
  executeCommand: (terminalId: string, command: string) => Promise<void>;

  // Toggle shell mode
  setShellMode: (terminalId: string, isShellMode: boolean) => void;
}

const generateTerminalId = () => {
  return `term-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
};

const createDefaultTerminal = (sessionId: string, terminalId: string): Terminal => ({
  id: terminalId,
  sessionId,
  name: `Terminal ${new Date().toLocaleTimeString()}`,
  entries: [],
  isExecuting: false,
  createdAt: new Date(),
  isShellMode: false,
});

export const useTerminalStore = create<TerminalState>()(
  persist(
    (set, get) => ({
      terminals: {},
      sessionTerminalIds: {},
      activeTerminalIds: {},

      getActiveTerminal: (sessionId: string) => {
        const state = get();
        const activeTerminalId = state.activeTerminalIds[sessionId];

        if (!activeTerminalId) {
          // Return null if no active terminal - component should create it
          return null;
        }

        return state.terminals[activeTerminalId] || null;
      },

      getSessionTerminals: (sessionId: string) => {
        const state = get();
        const terminalIds = state.sessionTerminalIds[sessionId] || [];
        return terminalIds
          .map((id) => state.terminals[id])
          .filter((t): t is Terminal => t !== undefined);
      },

  createTerminal: (sessionId: string, welcomeMessages?: string[]) => {
    const terminalId = generateTerminalId();
    const terminal = createDefaultTerminal(sessionId, terminalId);

    // Add welcome messages if provided
    if (welcomeMessages) {
      terminal.entries = welcomeMessages.map((content, index) => ({
        id: `welcome-${terminalId}-${index}`,
        type: 'system' as const,
        content,
        timestamp: new Date(),
      }));
    }

    set((state) => {
      const existingIds = state.sessionTerminalIds[sessionId] || [];
      return {
        terminals: {
          ...state.terminals,
          [terminalId]: terminal,
        },
        sessionTerminalIds: {
          ...state.sessionTerminalIds,
          [sessionId]: [...existingIds, terminalId],
        },
        activeTerminalIds: {
          ...state.activeTerminalIds,
          [sessionId]: terminalId,
        },
      };
    });

    return terminalId;
  },

  setActiveTerminal: (sessionId: string, terminalId: string) => {
    set((state) => ({
      activeTerminalIds: {
        ...state.activeTerminalIds,
        [sessionId]: terminalId,
      },
    }));
  },

  closeTerminal: (terminalId: string) => {
    set((state) => {
      const terminal = state.terminals[terminalId];
      if (!terminal) return state;

      const { sessionId } = terminal;
      const terminalIds = state.sessionTerminalIds[sessionId] || [];
      const newTerminalIds = terminalIds.filter((id) => id !== terminalId);

      // Remove terminal
      const newTerminals = { ...state.terminals };
      delete newTerminals[terminalId];

      // Update active terminal if the closed one was active
      let newActiveTerminalIds = { ...state.activeTerminalIds };
      if (state.activeTerminalIds[sessionId] === terminalId) {
        // Switch to another terminal or remove active terminal
        const nextTerminalId = newTerminalIds[newTerminalIds.length - 1];
        if (nextTerminalId) {
          newActiveTerminalIds[sessionId] = nextTerminalId;
        } else {
          delete newActiveTerminalIds[sessionId];
        }
      }

      return {
        terminals: newTerminals,
        sessionTerminalIds: {
          ...state.sessionTerminalIds,
          [sessionId]: newTerminalIds,
        },
        activeTerminalIds: newActiveTerminalIds,
      };
    });
  },

  addEntry: (terminalId: string, entry) => {
    const newEntry: TerminalEntry = {
      ...entry,
      id: `${entry.type}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      timestamp: new Date(),
    };

    set((state) => {
      const terminal = state.terminals[terminalId];
      if (!terminal) return state;

      return {
        terminals: {
          ...state.terminals,
          [terminalId]: {
            ...terminal,
            entries: [...terminal.entries, newEntry],
          },
        },
      };
    });
  },

  setExecuting: (terminalId: string, isExecuting: boolean) => {
    set((state) => {
      const terminal = state.terminals[terminalId];
      if (!terminal) return state;

      return {
        terminals: {
          ...state.terminals,
          [terminalId]: {
            ...terminal,
            isExecuting,
          },
        },
      };
    });
  },

  clearTerminal: (terminalId: string) => {
    set((state) => {
      const terminal = state.terminals[terminalId];
      if (!terminal) return state;

      return {
        terminals: {
          ...state.terminals,
          [terminalId]: {
            ...terminal,
            entries: [],
          },
        },
      };
    });
  },

  executeCommand: async (terminalId: string, command: string) => {
    const state = get();
    const terminal = state.terminals[terminalId];
    if (!terminal || terminal.isExecuting) return;

    // Handle exit command in shell mode
    if (terminal.isShellMode && command.trim().toLowerCase() === 'exit') {
      get().addEntry(terminalId, {
        type: 'input',
        content: command,
      });
      get().addEntry(terminalId, {
        type: 'system',
        content: 'Exiting interactive shell. Returning to virga.',
      });
      get().setShellMode(terminalId, false);
      return;
    }

    // Handle shell command to enter shell mode
    if (!terminal.isShellMode && command.trim().toLowerCase() === 'shell') {
      get().addEntry(terminalId, {
        type: 'input',
        content: command,
      });
      get().addEntry(terminalId, {
        type: 'system',
        content: "Starting interactive shell. Type 'exit' to return to virga.",
      });
      get().setShellMode(terminalId, true);
      return;
    }

    // Add command to entry
    get().addEntry(terminalId, {
      type: 'input',
      content: command,
    });

    // Set executing state
    get().setExecuting(terminalId, true);

    try {
      // Use real API - execute command through commandRouter
      const parsed = parseCommand(command);

      // Get session information
      const sessions = await api.sessions.getSessions();
      const session = sessions.find(s => s.id === terminal.sessionId);

      if (!session) {
        throw new Error(`Session ${terminal.sessionId} not found`);
      }

      // Execute command through commandRouter
      const result = await executeCommand(parsed, session);

      // Display result
      get().addEntry(terminalId, {
        type: result.isError ? 'error' : 'output',
        content: result.content,
      });
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'Command execution failed';
      get().addEntry(terminalId, {
        type: 'error',
        content: `Error: ${errorMessage}`,
      });
      console.error('Command execution error:', error);
    } finally {
      get().setExecuting(terminalId, false);
    }
  },

  setShellMode: (terminalId: string, isShellMode: boolean) => {
    set((state) => {
      const terminal = state.terminals[terminalId];
      if (!terminal) return state;

      return {
        terminals: {
          ...state.terminals,
          [terminalId]: {
            ...terminal,
            isShellMode,
          },
        },
      };
    });
  },
    }),
    {
      name: 'virga-terminal-storage',
      storage: zustandStorage,
      // Only persist data state, not methods
      partialize: (state) => ({
        terminals: state.terminals,
        sessionTerminalIds: state.sessionTerminalIds,
        activeTerminalIds: state.activeTerminalIds,
      }),
    }
  )
);
