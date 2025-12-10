import { useState, useEffect } from 'react';
import type { SecuritySettings } from '../../types/settings';
import type { SettingsSectionProps } from './types';

export const SecuritySettingsSection = ({
  settings,
  onUpdate,
  onSave,
}: SettingsSectionProps<SecuritySettings>) => {
  const [localSettings, setLocalSettings] = useState(settings);

  useEffect(() => {
    setLocalSettings(settings);
  }, [settings]);

  const handleChange = (field: keyof SecuritySettings, value: any) => {
    const updated = { ...localSettings, [field]: value };
    setLocalSettings(updated);
  };

  const handleBlur = async (field: keyof SecuritySettings) => {
    if (localSettings[field] !== settings[field]) {
      await onUpdate({ [field]: localSettings[field] });
      onSave();
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-dark-text-primary mb-4">
          Security Settings
        </h2>
        <p className="text-sm text-dark-text-secondary mb-6">
          Configure Web UI security and access management settings.
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Session Timeout (minutes)
          </label>
          <input
            type="number"
            value={localSettings.sessionTimeout}
            onChange={(e) => handleChange('sessionTimeout', parseInt(e.target.value))}
            onBlur={() => handleBlur('sessionTimeout')}
            min="5"
            max="1440"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Time until automatic logout when inactive
          </p>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Auto Logout
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Automatically logout on timeout
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.autoLogout}
              onChange={(e) => {
                handleChange('autoLogout', e.target.checked);
                handleBlur('autoLogout');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Two-Factor Authentication (2FA)
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Require additional authentication at login
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.require2FA}
              onChange={(e) => {
                handleChange('require2FA', e.target.checked);
                handleBlur('require2FA');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Remember Password
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Save password in browser
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.rememberPassword}
              onChange={(e) => {
                handleChange('rememberPassword', e.target.checked);
                handleBlur('rememberPassword');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="border-t border-gray-700 pt-4 mt-4">
          <h3 className="text-sm font-medium text-dark-text-primary mb-3">
            Audit Log
          </h3>

          <div className="flex items-center justify-between py-2">
            <div>
              <label className="text-sm font-medium text-dark-text-primary">
                Enable Audit Log
              </label>
              <p className="text-xs text-dark-text-secondary mt-1">
                Record all operations
              </p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={localSettings.auditLog}
                onChange={(e) => {
                  handleChange('auditLog', e.target.checked);
                  handleBlur('auditLog');
                }}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between py-2">
            <div>
              <label className="text-sm font-medium text-dark-text-primary">
                Log Commands
              </label>
              <p className="text-xs text-dark-text-secondary mt-1">
                Record executed commands in log
              </p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={localSettings.logCommands}
                onChange={(e) => {
                  handleChange('logCommands', e.target.checked);
                  handleBlur('logCommands');
                }}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </label>
          </div>
        </div>

        <div className="bg-yellow-500/10 border border-yellow-500/30 rounded p-3">
          <p className="text-xs text-yellow-400">
            Security settings changes affect the entire application. Please configure appropriately.
          </p>
        </div>
      </div>
    </div>
  );
};
