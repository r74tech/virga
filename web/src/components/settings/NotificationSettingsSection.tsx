import { useState, useEffect } from 'react';
import type { NotificationSettings } from '../../types/settings';
import type { SettingsSectionProps } from './types';

export const NotificationSettingsSection = ({
  settings,
  onUpdate,
  onSave,
}: SettingsSectionProps<NotificationSettings>) => {
  const [localSettings, setLocalSettings] = useState(settings);

  useEffect(() => {
    setLocalSettings(settings);
  }, [settings]);

  const handleChange = (field: keyof NotificationSettings, value: any) => {
    const updated = { ...localSettings, [field]: value };
    setLocalSettings(updated);
  };

  const handleBlur = async (field: keyof NotificationSettings) => {
    if (localSettings[field] !== settings[field]) {
      await onUpdate({ [field]: localSettings[field] });
      onSave();
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-dark-text-primary mb-4">
          Notification Settings
        </h2>
        <p className="text-sm text-dark-text-secondary mb-6">
          Manage notification methods for system events.
        </p>
      </div>

      <div className="space-y-4">
        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              New Session Alert
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Notify when a new session is established
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.newSessionAlert}
              onChange={(e) => {
                handleChange('newSessionAlert', e.target.checked);
                handleBlur('newSessionAlert');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Session Disconnect Alert
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Notify when a session is disconnected
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.sessionDisconnectAlert}
              onChange={(e) => {
                handleChange('sessionDisconnectAlert', e.target.checked);
                handleBlur('sessionDisconnectAlert');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Command Complete Alert
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Notify when command execution completes
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.commandCompleteAlert}
              onChange={(e) => {
                handleChange('commandCompleteAlert', e.target.checked);
                handleBlur('commandCompleteAlert');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Error Alert
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Notify when an error occurs
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.errorAlert}
              onChange={(e) => {
                handleChange('errorAlert', e.target.checked);
                handleBlur('errorAlert');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="border-t border-gray-700 pt-4 mt-4">
          <h3 className="text-sm font-medium text-dark-text-primary mb-3">
            Notification Methods
          </h3>

          <div className="flex items-center justify-between py-2">
            <div>
              <label className="text-sm font-medium text-dark-text-primary">
                Browser Notifications
              </label>
              <p className="text-xs text-dark-text-secondary mt-1">
                Use browser desktop notifications
              </p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={localSettings.browserNotifications}
                onChange={(e) => {
                  handleChange('browserNotifications', e.target.checked);
                  handleBlur('browserNotifications');
                }}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between py-2">
            <div>
              <label className="text-sm font-medium text-dark-text-primary">
                Notification Sound
              </label>
              <p className="text-xs text-dark-text-secondary mt-1">
                Play sound on notifications
              </p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={localSettings.soundEnabled}
                onChange={(e) => {
                  handleChange('soundEnabled', e.target.checked);
                  handleBlur('soundEnabled');
                }}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </label>
          </div>

          {localSettings.soundEnabled && (
            <div className="ml-4 mt-2">
              <label className="block text-sm font-medium text-dark-text-secondary mb-2">
                Volume
              </label>
              <input
                type="range"
                min="0"
                max="100"
                value={localSettings.soundVolume}
                onChange={(e) => handleChange('soundVolume', parseInt(e.target.value))}
                onMouseUp={() => handleBlur('soundVolume')}
                onTouchEnd={() => handleBlur('soundVolume')}
                className="w-full h-2 bg-gray-600 rounded-lg appearance-none cursor-pointer accent-primary-500"
              />
              <p className="text-xs text-dark-text-secondary mt-1">
                Current volume: {localSettings.soundVolume}%
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
