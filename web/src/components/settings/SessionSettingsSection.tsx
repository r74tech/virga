import { useState, useEffect } from 'react';
import type { SessionSettings } from '../../types/settings';
import type { SettingsSectionProps } from './types';

export const SessionSettingsSection = ({
  settings,
  onUpdate,
  onSave,
}: SettingsSectionProps<SessionSettings>) => {
  const [localSettings, setLocalSettings] = useState(settings);

  useEffect(() => {
    setLocalSettings(settings);
  }, [settings]);

  const handleChange = (field: keyof SessionSettings, value: number | boolean) => {
    const updated = { ...localSettings, [field]: value };
    setLocalSettings(updated);
  };

  const handleBlur = async (field: keyof SessionSettings) => {
    if (localSettings[field] !== settings[field]) {
      await onUpdate({ [field]: localSettings[field] });
      onSave();
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-dark-text-primary mb-4">
          Session Management Settings
        </h2>
        <p className="text-sm text-dark-text-secondary mb-6">
          Manage C2 session and beacon settings.
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2" htmlFor="defaultBeaconInterval">
            Default Beacon Interval (seconds)
          </label>
          <input
            type="number"
            value={localSettings.defaultBeaconInterval}
            onChange={(e) => handleChange('defaultBeaconInterval', Number.parseInt(e.target.value))}
            onBlur={() => handleBlur('defaultBeaconInterval')}
            min="1"
            max="3600"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Set how often the beacon checks in with the C2 server
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2" htmlFor="jitter">
            Jitter (%)
          </label>
          <input
            id="jitter"
            type="number"
            value={localSettings.jitter}
            onChange={(e) => handleChange('jitter', Number.parseInt(e.target.value))}
            onBlur={() => handleBlur('jitter')}
            min="0"
            max="100"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Add randomness to beacon interval to evade detection
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2" htmlFor="inactiveTimeout">
            Inactive Timeout (minutes)
          </label>
          <input
            id="inactiveTimeout"
            type="number"
            value={localSettings.inactiveTimeout}
            onChange={(e) => handleChange('inactiveTimeout', Number.parseInt(e.target.value))}
            onBlur={() => handleBlur('inactiveTimeout')}
            min="1"
            max="1440"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Time before marking session as inactive
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2" htmlFor="chatTimeout">
            Chat Timeout (minutes)
          </label>
          <input
            id="chatTimeout"
            type="number"
            value={localSettings.chatTimeout}
            onChange={(e) => handleChange('chatTimeout', Number.parseInt(e.target.value))}
            onBlur={() => handleBlur('chatTimeout')}
            min="1"
            max="60"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Maximum time to wait for AI chat response
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2" htmlFor="maxReconnectAttempts">
            Max Reconnect Attempts
          </label>
          <input
            id="maxReconnectAttempts"
            type="number"
            value={localSettings.maxReconnectAttempts}
            onChange={(e) => handleChange('maxReconnectAttempts', Number.parseInt(e.target.value))}
            onBlur={() => handleBlur('maxReconnectAttempts')}
            min="0"
            max="50"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Number of reconnection attempts after disconnect (0 for unlimited)
          </p>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary" htmlFor="autoReconnect">
              Auto Reconnect
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Automatically attempt to reconnect on disconnect
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              id="autoReconnect"
              type="checkbox"
              checked={localSettings.autoReconnect}
              onChange={(e) => {
                handleChange('autoReconnect', e.target.checked);
                handleBlur('autoReconnect');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600" />
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary" htmlFor="saveCommandHistory">
              Save Command History
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Save history of executed commands
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              id="saveCommandHistory"
              type="checkbox"
              checked={localSettings.saveCommandHistory}
              onChange={(e) => {
                handleChange('saveCommandHistory', e.target.checked);
                handleBlur('saveCommandHistory');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600" />
          </label>
        </div>
      </div>
    </div>
  );
};
