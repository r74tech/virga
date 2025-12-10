import { useState, useEffect } from 'react';
import type { ConnectionSettings } from '../../types/settings';
import type { SettingsSectionProps } from './types';

export const ConnectionSettingsSection = ({
  settings,
  onUpdate,
  onSave,
}: SettingsSectionProps<ConnectionSettings>) => {
  const [localSettings, setLocalSettings] = useState(settings);

  useEffect(() => {
    setLocalSettings(settings);
  }, [settings]);

  const handleChange = (field: keyof ConnectionSettings, value: any) => {
    const updated = { ...localSettings, [field]: value };
    setLocalSettings(updated);
  };

  const handleBlur = async (field: keyof ConnectionSettings) => {
    if (localSettings[field] !== settings[field]) {
      await onUpdate({ [field]: localSettings[field] });
      onSave();
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-dark-text-primary mb-4">
          C2 Server Connection Settings
        </h2>
        <p className="text-sm text-dark-text-secondary mb-6">
          Manage settings related to C2 server connections.
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Environment
          </label>
          <select
            value={localSettings.environment}
            onChange={(e) =>
              handleChange('environment', e.target.value as 'development' | 'staging' | 'production')
            }
            onBlur={() => handleBlur('environment')}
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          >
            <option value="development">Development</option>
            <option value="staging">Staging</option>
            <option value="production">Production</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            API Endpoint
          </label>
          <input
            type="text"
            value={localSettings.apiEndpoint}
            onChange={(e) => handleChange('apiEndpoint', e.target.value)}
            onBlur={() => handleBlur('apiEndpoint')}
            placeholder="http://localhost:8080/api"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary placeholder-dark-text-secondary focus:outline-none focus:border-primary-500"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            WebSocket Endpoint
          </label>
          <input
            type="text"
            value={localSettings.websocketEndpoint}
            onChange={(e) => handleChange('websocketEndpoint', e.target.value)}
            onBlur={() => handleBlur('websocketEndpoint')}
            placeholder="ws://localhost:8080/ws"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary placeholder-dark-text-secondary focus:outline-none focus:border-primary-500"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Authentication Token (Optional)
          </label>
          <input
            type="password"
            value={localSettings.authToken ?? ''}
            onChange={(e) => handleChange('authToken', e.target.value === '' ? undefined : e.target.value)}
            onBlur={() => handleBlur('authToken')}
            placeholder="Enter authentication token"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary placeholder-dark-text-secondary focus:outline-none focus:border-primary-500"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Timeout (milliseconds)
          </label>
          <input
            type="number"
            value={localSettings.timeout}
            onChange={(e) => handleChange('timeout', parseInt(e.target.value))}
            onBlur={() => handleBlur('timeout')}
            min="1000"
            max="120000"
            step="1000"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Current setting: {localSettings.timeout / 1000} seconds
          </p>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-dark-text-primary mb-2">
              Retry Attempts
            </label>
            <input
              type="number"
              value={localSettings.retryAttempts}
              onChange={(e) => handleChange('retryAttempts', parseInt(e.target.value))}
              onBlur={() => handleBlur('retryAttempts')}
              min="0"
              max="10"
              className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-dark-text-primary mb-2">
              Retry Delay (milliseconds)
            </label>
            <input
              type="number"
              value={localSettings.retryDelay}
              onChange={(e) => handleChange('retryDelay', parseInt(e.target.value))}
              onBlur={() => handleBlur('retryDelay')}
              min="100"
              max="10000"
              step="100"
              className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
            />
          </div>
        </div>
      </div>
    </div>
  );
};
