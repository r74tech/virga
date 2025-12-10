import { create } from 'zustand';
import type { Settings, ConnectionSettings, UISettings, SessionSettings, NotificationSettings, TerminalSettings, SecuritySettings, ForestMapSettings } from '../types/settings';
import { DEFAULT_SETTINGS } from '../types/settings';
import { saveSettings, loadSettings } from '../utils/db';

interface SettingsState {
  settings: Settings;
  isLoaded: boolean;

  loadSettings: () => Promise<void>;
  updateConnectionSettings: (settings: Partial<ConnectionSettings>) => Promise<void>;
  updateUISettings: (settings: Partial<UISettings>) => Promise<void>;
  updateSessionSettings: (settings: Partial<SessionSettings>) => Promise<void>;
  updateNotificationSettings: (settings: Partial<NotificationSettings>) => Promise<void>;
  updateTerminalSettings: (settings: Partial<TerminalSettings>) => Promise<void>;
  updateSecuritySettings: (settings: Partial<SecuritySettings>) => Promise<void>;
  updateForestMapSettings: (settings: Partial<ForestMapSettings>) => Promise<void>;
  resetToDefaults: () => Promise<void>;
}

export const useSettingsStore = create<SettingsState>((set, get) => ({
  settings: DEFAULT_SETTINGS,
  isLoaded: false,

  loadSettings: async () => {
    const loaded = await loadSettings();
    if (loaded) {
      // Merge with defaults to ensure all fields exist
      const merged: Settings = {
        connection: { ...DEFAULT_SETTINGS.connection, ...loaded.connection },
        ui: { ...DEFAULT_SETTINGS.ui, ...loaded.ui },
        session: { ...DEFAULT_SETTINGS.session, ...loaded.session },
        notification: { ...DEFAULT_SETTINGS.notification, ...loaded.notification },
        terminal: { ...DEFAULT_SETTINGS.terminal, ...loaded.terminal },
        security: { ...DEFAULT_SETTINGS.security, ...loaded.security },
        forestMap: { ...DEFAULT_SETTINGS.forestMap, ...loaded.forestMap },
      };
      set({ settings: merged, isLoaded: true });
      // Save merged settings to ensure all new fields are persisted
      await saveSettings(merged);
    } else {
      set({ settings: DEFAULT_SETTINGS, isLoaded: true });
      await saveSettings(DEFAULT_SETTINGS);
    }
  },

  updateConnectionSettings: async (newSettings) => {
    const { settings } = get();
    const updated = {
      ...settings,
      connection: {
        ...settings.connection,
        ...newSettings,
      },
    };
    set({ settings: updated });
    await saveSettings(updated);
  },

  updateUISettings: async (newSettings) => {
    const { settings } = get();
    const updated = {
      ...settings,
      ui: {
        ...settings.ui,
        ...newSettings,
      },
    };
    set({ settings: updated });
    await saveSettings(updated);
  },

  updateSessionSettings: async (newSettings) => {
    const { settings } = get();
    const updated = {
      ...settings,
      session: {
        ...settings.session,
        ...newSettings,
      },
    };
    set({ settings: updated });
    await saveSettings(updated);
  },

  updateNotificationSettings: async (newSettings) => {
    const { settings } = get();
    const updated = {
      ...settings,
      notification: {
        ...settings.notification,
        ...newSettings,
      },
    };
    set({ settings: updated });
    await saveSettings(updated);
  },

  updateTerminalSettings: async (newSettings) => {
    const { settings } = get();
    const updated = {
      ...settings,
      terminal: {
        ...settings.terminal,
        ...newSettings,
      },
    };
    set({ settings: updated });
    await saveSettings(updated);
  },

  updateSecuritySettings: async (newSettings) => {
    const { settings } = get();
    const updated = {
      ...settings,
      security: {
        ...settings.security,
        ...newSettings,
      },
    };
    set({ settings: updated });
    await saveSettings(updated);
  },

  updateForestMapSettings: async (newSettings) => {
    const { settings } = get();
    const updated = {
      ...settings,
      forestMap: {
        ...settings.forestMap,
        ...newSettings,
      },
    };
    set({ settings: updated });
    await saveSettings(updated);
  },

  resetToDefaults: async () => {
    set({ settings: DEFAULT_SETTINGS });
    await saveSettings(DEFAULT_SETTINGS);
  },
}));
