import { useState, useEffect } from 'react';
import type { UISettings } from '../../types/settings';
import type { SettingsSectionProps } from './types';

export const UISettingsSection = ({
  settings,
  onUpdate,
  onSave,
}: SettingsSectionProps<UISettings>) => {
  const [localSettings, setLocalSettings] = useState(settings);

  useEffect(() => {
    setLocalSettings(settings);
  }, [settings]);

  const handleChange = (field: keyof UISettings, value: any) => {
    const updated = { ...localSettings, [field]: value };
    setLocalSettings(updated);
  };

  const handleBlur = async (field: keyof UISettings) => {
    if (localSettings[field] !== settings[field]) {
      await onUpdate({ [field]: localSettings[field] });
      onSave();
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-dark-text-primary mb-4">
          UI Settings
        </h2>
        <p className="text-sm text-dark-text-secondary mb-6">
          Manage interface display settings.
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Theme
          </label>
          <select
            value={localSettings.theme}
            onChange={(e) => handleChange('theme', e.target.value as 'light' | 'dark' | 'auto')}
            onBlur={() => handleBlur('theme')}
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          >
            <option value="light">Light Mode</option>
            <option value="dark">Dark Mode</option>
            <option value="auto">Follow System</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Language
          </label>
          <select
            value={localSettings.language}
            onChange={(e) => handleChange('language', e.target.value as 'en' | 'ja')}
            onBlur={() => handleBlur('language')}
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          >
            <option value="ja">Japanese (Japanese)</option>
            <option value="en">English</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Timezone
          </label>
          <select
            value={localSettings.timezone}
            onChange={(e) => handleChange('timezone', e.target.value)}
            onBlur={() => handleBlur('timezone')}
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          >
            <option value="Asia/Tokyo">Asia/Tokyo (JST)</option>
            <option value="UTC">Coordinated Universal Time (UTC)</option>
            <option value="America/New_York">America/New York (EST)</option>
            <option value="America/Los_Angeles">America/Los Angeles (PST)</option>
            <option value="Europe/London">Europe/London (GMT)</option>
            <option value="Australia/Sydney">Australia/Sydney (AEDT)</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Date Format
          </label>
          <select
            value={localSettings.dateFormat}
            onChange={(e) => handleChange('dateFormat', e.target.value)}
            onBlur={() => handleBlur('dateFormat')}
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          >
            <option value="YYYY/MM/DD">YYYY/MM/DD (2025/10/28)</option>
            <option value="MM/DD/YYYY">MM/DD/YYYY (10/28/2025)</option>
            <option value="DD/MM/YYYY">DD/MM/YYYY (28/10/2025)</option>
            <option value="YYYY-MM-DD">YYYY-MM-DD (2025-10-28)</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Time Format
          </label>
          <select
            value={localSettings.timeFormat}
            onChange={(e) => handleChange('timeFormat', e.target.value as '12h' | '24h')}
            onBlur={() => handleBlur('timeFormat')}
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          >
            <option value="24h">24-hour (14:30)</option>
            <option value="12h">12-hour (2:30 PM)</option>
          </select>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Animation Effects
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Enable UI animations
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.animations}
              onChange={(e) => {
                handleChange('animations', e.target.checked);
                handleBlur('animations');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Compact Mode
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Display more information
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.compactMode}
              onChange={(e) => {
                handleChange('compactMode', e.target.checked);
                handleBlur('compactMode');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>
      </div>
    </div>
  );
};
