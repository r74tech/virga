import { useState, useEffect } from 'react';
import { Settings as SettingsIcon } from 'lucide-react';
import toast from 'react-hot-toast';
import { useSettingsStore } from '../stores/settingsStore';
import { ConnectionSettingsSection } from '../components/settings/ConnectionSettingsSection';
import { UISettingsSection } from '../components/settings/UISettingsSection';
import { SessionSettingsSection } from '../components/settings/SessionSettingsSection';
import { NotificationSettingsSection } from '../components/settings/NotificationSettingsSection';
import { TerminalSettingsSection } from '../components/settings/TerminalSettingsSection';
import { SecuritySettingsSection } from '../components/settings/SecuritySettingsSection';
import { ForestMapSettingsSection } from '../components/settings/ForestMapSettingsSection';
import { DataManagementSection } from '../components/settings/DataManagementSection';
import type { Settings } from '../types/settings';

type SettingsTab =
  | 'connection'
  | 'ui'
  | 'session'
  | 'notification'
  | 'terminal'
  | 'security'
  | 'forestMap'
  | 'data';

const tabs: { id: SettingsTab; label: string }[] = [
  { id: 'connection', label: 'C2 Server Connection' },
  { id: 'ui', label: 'UI & Display' },
  { id: 'session', label: 'Session Management' },
  { id: 'notification', label: 'Notifications' },
  { id: 'terminal', label: 'Terminal' },
  { id: 'security', label: 'Security' },
  { id: 'forestMap', label: 'Forest Map' },
  { id: 'data', label: 'Data Management' },
];

export const SettingsPage = () => {
  const [activeTab, setActiveTab] = useState<SettingsTab>('connection');
  const {
    settings,
    isLoaded,
    loadSettings,
    updateConnectionSettings,
    updateUISettings,
    updateSessionSettings,
    updateNotificationSettings,
    updateTerminalSettings,
    updateSecuritySettings,
    updateForestMapSettings,
    resetToDefaults,
  } = useSettingsStore();

  useEffect(() => {
    if (!isLoaded) {
      loadSettings();
    }
  }, [isLoaded, loadSettings]);

  const handleSave = () => {
    // Settings are saved automatically by each section's onUpdate
    toast.success('Settings saved');
  };

  const handleImport = async (importedSettings: Settings) => {
    try {
      await updateConnectionSettings(importedSettings.connection);
      await updateUISettings(importedSettings.ui);
      await updateSessionSettings(importedSettings.session);
      await updateNotificationSettings(importedSettings.notification);
      await updateTerminalSettings(importedSettings.terminal);
      await updateSecuritySettings(importedSettings.security);
      await updateForestMapSettings(importedSettings.forestMap);
      toast.success('Settings imported');
    } catch (error) {
      toast.error('Failed to import settings');
    }
  };

  if (!isLoaded) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-dark-text-secondary">Loading settings...</div>
      </div>
    );
  }

  return (
    <div className="flex h-full bg-dark-bg">
      {/* Sidebar */}
      <div className="w-64 bg-dark-bg-secondary border-r border-gray-700 overflow-y-auto">
        <div className="p-4 border-b border-gray-700">
          <div className="flex items-center gap-2">
            <SettingsIcon className="w-5 h-5 text-primary-500" />
            <h1 className="text-lg font-semibold text-dark-text-primary">Settings</h1>
          </div>
        </div>
        <nav className="p-2">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={`w-full text-left px-3 py-2 rounded mb-1 transition-colors ${
                activeTab === tab.id
                  ? 'bg-primary-600 text-white'
                  : 'text-dark-text-primary hover:bg-dark-bg-tertiary'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto p-6">
          {activeTab === 'connection' && (
            <ConnectionSettingsSection
              settings={settings.connection}
              onUpdate={updateConnectionSettings}
              onSave={handleSave}
            />
          )}
          {activeTab === 'ui' && (
            <UISettingsSection
              settings={settings.ui}
              onUpdate={updateUISettings}
              onSave={handleSave}
            />
          )}
          {activeTab === 'session' && (
            <SessionSettingsSection
              settings={settings.session}
              onUpdate={updateSessionSettings}
              onSave={handleSave}
            />
          )}
          {activeTab === 'notification' && (
            <NotificationSettingsSection
              settings={settings.notification}
              onUpdate={updateNotificationSettings}
              onSave={handleSave}
            />
          )}
          {activeTab === 'terminal' && (
            <TerminalSettingsSection
              settings={settings.terminal}
              onUpdate={updateTerminalSettings}
              onSave={handleSave}
            />
          )}
          {activeTab === 'security' && (
            <SecuritySettingsSection
              settings={settings.security}
              onUpdate={updateSecuritySettings}
              onSave={handleSave}
            />
          )}
          {activeTab === 'forestMap' && (
            <ForestMapSettingsSection
              settings={settings.forestMap}
              onUpdate={updateForestMapSettings}
              onSave={handleSave}
            />
          )}
          {activeTab === 'data' && (
            <DataManagementSection
              settings={settings}
              onImport={handleImport}
              onReset={resetToDefaults}
            />
          )}
        </div>
      </div>
    </div>
  );
};
