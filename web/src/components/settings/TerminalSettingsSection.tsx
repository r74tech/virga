import { useState, useEffect } from 'react';
import type { TerminalSettings } from '../../types/settings';
import type { SettingsSectionProps } from './types';

export const TerminalSettingsSection = ({
  settings,
  onUpdate,
  onSave,
}: SettingsSectionProps<TerminalSettings>) => {
  const [localSettings, setLocalSettings] = useState(settings);

  useEffect(() => {
    setLocalSettings(settings);
  }, [settings]);

  const handleChange = (field: keyof TerminalSettings, value: any) => {
    const updated = { ...localSettings, [field]: value };
    setLocalSettings(updated);
  };

  const handleBlur = async (field: keyof TerminalSettings) => {
    if (localSettings[field] !== settings[field]) {
      await onUpdate({ [field]: localSettings[field] });
      onSave();
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-dark-text-primary mb-4">
          Terminal Settings
        </h2>
        <p className="text-sm text-dark-text-secondary mb-6">
          Manage terminal display and behavior settings.
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Font Family
          </label>
          <select
            value={localSettings.fontFamily}
            onChange={(e) => handleChange('fontFamily', e.target.value)}
            onBlur={() => handleBlur('fontFamily')}
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500 font-mono"
          >
            <option value="JetBrains Mono, monospace">JetBrains Mono</option>
            <option value="Fira Code, monospace">Fira Code</option>
            <option value="Consolas, monospace">Consolas</option>
            <option value="Monaco, monospace">Monaco</option>
            <option value="Courier New, monospace">Courier New</option>
            <option value="monospace">Monospace</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Font Size (px)
          </label>
          <input
            type="number"
            value={localSettings.fontSize}
            onChange={(e) => handleChange('fontSize', parseInt(e.target.value))}
            onBlur={() => handleBlur('fontSize')}
            min="8"
            max="24"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Color Scheme
          </label>
          <select
            value={localSettings.colorScheme}
            onChange={(e) => handleChange('colorScheme', e.target.value)}
            onBlur={() => handleBlur('colorScheme')}
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          >
            <option value="default">Default</option>
            <option value="dracula">Dracula</option>
            <option value="monokai">Monokai</option>
            <option value="solarized-dark">Solarized Dark</option>
            <option value="solarized-light">Solarized Light</option>
            <option value="nord">Nord</option>
            <option value="gruvbox">Gruvbox</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            History Size
          </label>
          <input
            type="number"
            value={localSettings.historySize}
            onChange={(e) => handleChange('historySize', parseInt(e.target.value))}
            onBlur={() => handleBlur('historySize')}
            min="50"
            max="10000"
            step="50"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Number of command history lines to retain in terminal
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Scrollback Lines
          </label>
          <input
            type="number"
            value={localSettings.scrollback}
            onChange={(e) => handleChange('scrollback', parseInt(e.target.value))}
            onBlur={() => handleBlur('scrollback')}
            min="100"
            max="50000"
            step="100"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Terminal output buffer size
          </p>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Cursor Blink
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Blink terminal cursor
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.cursorBlink}
              onChange={(e) => {
                handleChange('cursorBlink', e.target.checked);
                handleBlur('cursorBlink');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Show Line Numbers
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Display line numbers in output
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.lineNumbers}
              onChange={(e) => {
                handleChange('lineNumbers', e.target.checked);
                handleBlur('lineNumbers');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Autocomplete
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Enable command autocomplete
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.autocomplete}
              onChange={(e) => {
                handleChange('autocomplete', e.target.checked);
                handleBlur('autocomplete');
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
