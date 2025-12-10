import { useState, useEffect } from 'react';
import type { ForestMapSettings } from '../../types/settings';
import type { SettingsSectionProps } from './types';

export const ForestMapSettingsSection = ({
  settings,
  onUpdate,
  onSave,
}: SettingsSectionProps<ForestMapSettings>) => {
  const [localSettings, setLocalSettings] = useState(settings);

  useEffect(() => {
    setLocalSettings(settings);
  }, [settings]);

  const handleChange = (field: keyof ForestMapSettings, value: any) => {
    const updated = { ...localSettings, [field]: value };
    setLocalSettings(updated);
  };

  const handleBlur = async (field: keyof ForestMapSettings) => {
    if (localSettings[field] !== settings[field]) {
      await onUpdate({ [field]: localSettings[field] });
      onSave();
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-dark-text-primary mb-4">
          Forest Map Settings
        </h2>
        <p className="text-sm text-dark-text-secondary mb-6">
          Manage network graph display and layout settings.
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Layout Algorithm
          </label>
          <select
            value={localSettings.layout}
            onChange={(e) => handleChange('layout', e.target.value as ForestMapSettings['layout'])}
            onBlur={() => handleBlur('layout')}
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          >
            <option value="cola">Cola (Physics Simulation)</option>
            <option value="cose">COSE (Force-Directed)</option>
            <option value="grid">Grid</option>
            <option value="circle">Circle</option>
            <option value="breadthfirst">Breadth First</option>
            <option value="concentric">Concentric</option>
          </select>
          <p className="text-xs text-dark-text-secondary mt-1">
            Select node placement method
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Node Spacing
          </label>
          <input
            type="number"
            value={localSettings.nodeSpacing}
            onChange={(e) => handleChange('nodeSpacing', parseInt(e.target.value))}
            onBlur={() => handleBlur('nodeSpacing')}
            min="20"
            max="500"
            step="10"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Ideal distance between nodes (pixels)
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Edge Length
          </label>
          <input
            type="number"
            value={localSettings.edgeLength}
            onChange={(e) => handleChange('edgeLength', parseInt(e.target.value))}
            onBlur={() => handleBlur('edgeLength')}
            min="50"
            max="500"
            step="10"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Ideal edge length (pixels)
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-dark-text-primary mb-2">
            Animation Duration (ms)
          </label>
          <input
            type="number"
            value={localSettings.animationDuration}
            onChange={(e) => handleChange('animationDuration', parseInt(e.target.value))}
            onBlur={() => handleBlur('animationDuration')}
            min="0"
            max="2000"
            step="100"
            className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-dark-text-secondary mt-1">
            Animation time for layout changes (0 to disable)
          </p>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Show Node Labels
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Display labels on nodes
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.showLabels}
              onChange={(e) => {
                handleChange('showLabels', e.target.checked);
                handleBlur('showLabels');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Show Edge Labels
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Display labels on edges
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.showEdgeLabels}
              onChange={(e) => {
                handleChange('showEdgeLabels', e.target.checked);
                handleBlur('showEdgeLabels');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Auto Center on Zoom
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Automatically center when selecting nodes
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.autoCenter}
              onChange={(e) => {
                handleChange('autoCenter', e.target.checked);
                handleBlur('autoCenter');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <label className="text-sm font-medium text-dark-text-primary">
              Auto Refresh
            </label>
            <p className="text-xs text-dark-text-secondary mt-1">
              Periodically update the graph
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.autoRefresh}
              onChange={(e) => {
                handleChange('autoRefresh', e.target.checked);
                handleBlur('autoRefresh');
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
        </div>

        {localSettings.autoRefresh && (
          <div className="ml-4">
            <label className="block text-sm font-medium text-dark-text-secondary mb-2">
              Refresh Interval (seconds)
            </label>
            <input
              type="number"
              value={localSettings.refreshInterval}
              onChange={(e) => handleChange('refreshInterval', parseInt(e.target.value))}
              onBlur={() => handleBlur('refreshInterval')}
              min="5"
              max="300"
              step="5"
              className="w-full bg-dark-bg-secondary border border-gray-600 rounded px-3 py-2 text-dark-text-primary focus:outline-none focus:border-primary-500"
            />
          </div>
        )}
      </div>
    </div>
  );
};
