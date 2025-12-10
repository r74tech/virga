import { useState } from 'react';
import { Download, Upload, Trash2 } from 'lucide-react';
import type { Settings } from '../../types/settings';
import { clearAllData } from '../../utils/db';

interface DataManagementSectionProps {
  settings: Settings;
  onImport: (settings: Settings) => Promise<void>;
  onReset: () => Promise<void>;
}

export const DataManagementSection = ({
  settings,
  onImport,
  onReset,
}: DataManagementSectionProps) => {
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [showResetConfirm, setShowResetConfirm] = useState(false);

  const handleExport = () => {
    const dataStr = JSON.stringify(settings, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `virga-settings-${new Date().toISOString().split('T')[0]}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleImport = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = 'application/json';
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;

      try {
        const text = await file.text();
        const importedSettings = JSON.parse(text);
        await onImport(importedSettings);
        alert('Settings imported successfully');
      } catch (error) {
        alert('Failed to import settings');
        console.error('Import error:', error);
      }
    };
    input.click();
  };

  const handleClearData = async () => {
    try {
      await clearAllData();
      setShowClearConfirm(false);
      alert('All data has been deleted');
    } catch (error) {
      alert('Failed to delete data');
      console.error('Clear data error:', error);
    }
  };

  const handleReset = async () => {
    try {
      await onReset();
      setShowResetConfirm(false);
      alert('Settings have been reset');
    } catch (error) {
      alert('Failed to reset settings');
      console.error('Reset error:', error);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-dark-text-primary mb-4">
          Data Management
        </h2>
        <p className="text-sm text-dark-text-secondary mb-6">
          Import, export, and reset settings and data.
        </p>
      </div>

      <div className="space-y-4">
        <div className="bg-dark-bg-secondary border border-gray-700 rounded-lg p-4">
          <h3 className="text-sm font-medium text-dark-text-primary mb-2">
            Export Settings
          </h3>
          <p className="text-xs text-dark-text-secondary mb-3">
            Download current settings as a JSON file
          </p>
          <button
            type="button"
            onClick={handleExport}
            className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded transition-colors"
          >
            <Download className="w-4 h-4" />
            Export Settings
          </button>
        </div>

        <div className="bg-dark-bg-secondary border border-gray-700 rounded-lg p-4">
          <h3 className="text-sm font-medium text-dark-text-primary mb-2">
            Import Settings
          </h3>
          <p className="text-xs text-dark-text-secondary mb-3">
            Load previously exported settings file
          </p>
          <button
            type="button"
            onClick={handleImport}
            className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded transition-colors"
          >
            <Upload className="w-4 h-4" />
            Import Settings
          </button>
        </div>

        <div className="bg-dark-bg-secondary border border-gray-700 rounded-lg p-4">
          <h3 className="text-sm font-medium text-dark-text-primary mb-2">
            Clear All Data
          </h3>
          <p className="text-xs text-dark-text-secondary mb-3">
            Delete chat history, conversations, and all other data (settings will be preserved)
          </p>
          {!showClearConfirm ? (
            <button
              type="button"
              onClick={() => setShowClearConfirm(true)}
              className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded transition-colors"
            >
              <Trash2 className="w-4 h-4" />
              Clear All Data
            </button>
          ) : (
            <div className="space-y-2">
              <p className="text-sm text-red-400">Are you sure you want to delete all data?</p>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={handleClearData}
                  className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded transition-colors"
                >
                  Delete
                </button>
                <button
                  type="button"
                  onClick={() => setShowClearConfirm(false)}
                  className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded transition-colors"
                >
                  Cancel
                </button>
              </div>
            </div>
          )}
        </div>

        <div className="bg-dark-bg-secondary border border-gray-700 rounded-lg p-4">
          <h3 className="text-sm font-medium text-dark-text-primary mb-2">
            Reset Settings
          </h3>
          <p className="text-xs text-dark-text-secondary mb-3">
            Reset all settings to default values
          </p>
          {!showResetConfirm ? (
            <button
              type="button"
              onClick={() => setShowResetConfirm(true)}
              className="flex items-center gap-2 px-4 py-2 bg-yellow-600 hover:bg-yellow-700 text-white rounded transition-colors"
            >
              Reset Settings
            </button>
          ) : (
            <div className="space-y-2">
              <p className="text-sm text-yellow-400">Are you sure you want to reset all settings?</p>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={handleReset}
                  className="px-4 py-2 bg-yellow-600 hover:bg-yellow-700 text-white rounded transition-colors"
                >
                  Reset
                </button>
                <button
                  type="button"
                  onClick={() => setShowResetConfirm(false)}
                  className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded transition-colors"
                >
                  Cancel
                </button>
              </div>
            </div>
          )}
        </div>

        <div className="bg-red-500/10 border border-red-500/30 rounded p-3">
          <p className="text-xs text-red-400">
            Data deletion and settings reset cannot be undone. We recommend exporting before proceeding.
          </p>
        </div>
      </div>
    </div>
  );
};
