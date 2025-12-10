import { useState } from 'react';
import { Plus, AlertCircle, Loader2, Radio, Square } from 'lucide-react';
import { useListenerStore } from '../stores/listenerStore';
import { useAutoRefresh } from '../hooks/useAutoRefresh';
import type { Listener, CreateListenerPayload, ListenerType } from '../types/listener';

export const ListenersPage = () => {
  const listeners = useListenerStore((state) => state.listeners);
  const selectedListener = useListenerStore((state) => state.selectedListener);
  const setSelectedListener = useListenerStore((state) => state.setSelectedListener);
  const stats = useListenerStore((state) => state.stats);
  const isLoading = useListenerStore((state) => state.isLoading);
  const error = useListenerStore((state) => state.error);
  const fetchListeners = useListenerStore((state) => state.fetchListeners);
  const createListener = useListenerStore((state) => state.createListener);
  const deleteListener = useListenerStore((state) => state.deleteListener);
  const clearError = useListenerStore((state) => state.clearError);

  const [showCreateForm, setShowCreateForm] = useState(false);
  const [formData, setFormData] = useState<CreateListenerPayload>({
    name: '',
    type: 'http',
    bindAddress: '0.0.0.0',
    port: 8080,
    tlsEnabled: false,
    uriPath: 'api/beacon',
    encryptionKey: '',
  });

  // Auto-refresh listeners every 10 seconds
  useAutoRefresh(fetchListeners, {
    enabled: true,
    interval: 10000,
  });

  const generateEncryptionKey = () => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*';
    const length = 32;
    let key = '';
    for (let i = 0; i < length; i++) {
      key += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setFormData({ ...formData, encryptionKey: key });
  };

  const handleCreateListener = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createListener(formData);
      setShowCreateForm(false);
      setFormData({
        name: '',
        type: 'http',
        bindAddress: '0.0.0.0',
        port: 8080,
        tlsEnabled: false,
        uriPath: 'api/beacon',
        encryptionKey: '',
      });
    } catch (error) {
      // Error is handled by the store
    }
  };

  const handleDeleteListener = async (name: string) => {
    if (window.confirm(`Are you sure you want to delete listener "${name}"?`)) {
      try {
        await deleteListener(name);
      } catch (error) {
        // Error is handled by the store
      }
    }
  };

  const getStatusColor = (status: Listener['status']) => {
    switch (status) {
      case 'running':
        return 'text-green-400';
      case 'stopped':
        return 'text-gray-400';
      case 'error':
        return 'text-red-400';
      default:
        return 'text-gray-400';
    }
  };

  const getStatusIcon = (status: Listener['status']) => {
    switch (status) {
      case 'running':
        return <Radio className="w-4 h-4" />;
      case 'stopped':
        return <Square className="w-4 h-4" />;
      case 'error':
        return <AlertCircle className="w-4 h-4" />;
      default:
        return <Square className="w-4 h-4" />;
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-gray-700 flex-shrink-0">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-2xl font-bold text-dark-text-primary">Listeners</h1>
          <div className="flex gap-4 items-center">
            <div className="flex gap-4 text-sm">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-green-500" />
                <span className="text-dark-text-secondary">
                  Running: {stats.runningListeners}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-gray-500" />
                <span className="text-dark-text-secondary">
                  Stopped: {stats.stoppedListeners}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-red-500" />
                <span className="text-dark-text-secondary">
                  Error: {stats.errorListeners}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-dark-text-secondary">
                  Total: {stats.totalListeners}
                </span>
              </div>
            </div>
            <button
              type="button"
              onClick={() => setShowCreateForm(true)}
              className="flex items-center gap-2 px-4 py-2 bg-primary-500 hover:bg-primary-600 text-white rounded-lg transition-colors"
            >
              <Plus className="w-4 h-4" />
              New Listener
            </button>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {/* Error Display */}
        {error && (
          <div className="mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg">
            <div className="flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <p className="text-red-400 text-sm font-medium mb-1">
                  Failed to load listeners
                </p>
                <p className="text-red-300 text-xs">{error}</p>
                <button
                  type="button"
                  onClick={() => {
                    clearError();
                    fetchListeners();
                  }}
                  className="mt-2 text-xs text-red-400 hover:text-red-300 underline"
                >
                  Retry
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Loading Display */}
        {isLoading && listeners.length === 0 ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <Loader2 className="w-8 h-8 text-primary-500 animate-spin mx-auto mb-2" />
              <p className="text-dark-text-secondary">Loading listeners...</p>
            </div>
          </div>
        ) : listeners.length === 0 ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <p className="text-dark-text-secondary text-lg mb-2">
                No listeners configured
              </p>
              <p className="text-dark-text-secondary text-sm">
                Click "New Listener" to create one
              </p>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {listeners.map((listener) => (
              <div
                key={listener.name}
                className={`p-4 rounded-lg border transition-all cursor-pointer ${
                  selectedListener?.name === listener.name
                    ? 'border-primary-500 bg-primary-500/10'
                    : 'border-gray-700 bg-dark-bg-secondary hover:border-gray-600'
                }`}
                onClick={() => setSelectedListener(listener)}
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <span className={getStatusColor(listener.status)}>
                      {getStatusIcon(listener.status)}
                    </span>
                    <h3 className="text-lg font-semibold text-dark-text-primary">
                      {listener.name}
                    </h3>
                  </div>
                  <span className="px-2 py-1 text-xs rounded bg-dark-bg-tertiary text-dark-text-secondary">
                    {listener.type.toUpperCase()}
                  </span>
                </div>

                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-dark-text-secondary">Address:</span>
                    <span className="text-dark-text-primary font-mono">
                      {listener.bindAddress}:{listener.port}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-dark-text-secondary">Status:</span>
                    <span className={getStatusColor(listener.status)}>
                      {listener.status}
                    </span>
                  </div>
                  {listener.metadata?.tlsEnabled && (
                    <div className="flex justify-between">
                      <span className="text-dark-text-secondary">TLS:</span>
                      <span className="text-green-400">Enabled</span>
                    </div>
                  )}
                  {listener.metadata?.connections !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-dark-text-secondary">Connections:</span>
                      <span className="text-dark-text-primary">
                        {listener.metadata.connections}
                      </span>
                    </div>
                  )}
                </div>

                <div className="mt-4 pt-4 border-t border-gray-700">
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeleteListener(listener.name);
                    }}
                    className="w-full px-3 py-1.5 text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors"
                  >
                    Delete Listener
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Create Listener Modal */}
      {showCreateForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-dark-bg-secondary border border-gray-700 rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-bold text-dark-text-primary mb-4">
              Create New Listener
            </h2>
            <form onSubmit={handleCreateListener} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-dark-text-secondary mb-1">
                  Name
                </label>
                <input
                  type="text"
                  required
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 bg-dark-bg-tertiary border border-gray-700 rounded text-dark-text-primary focus:outline-none focus:border-primary-500"
                  placeholder="my-listener"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-text-secondary mb-1">
                  Type
                </label>
                <select
                  value={formData.type}
                  onChange={(e) =>
                    setFormData({ ...formData, type: e.target.value as ListenerType })
                  }
                  className="w-full px-3 py-2 bg-dark-bg-tertiary border border-gray-700 rounded text-dark-text-primary focus:outline-none focus:border-primary-500"
                >
                  <option value="http">HTTP</option>
                  <option value="https">HTTPS</option>
                  {/* Unimplemented listener types */}
                  {/* <option value="tcp">TCP</option> */}
                  {/* <option value="smb">SMB</option> */}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-text-secondary mb-1">
                  Bind Address
                </label>
                <input
                  type="text"
                  required
                  value={formData.bindAddress}
                  onChange={(e) =>
                    setFormData({ ...formData, bindAddress: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-dark-bg-tertiary border border-gray-700 rounded text-dark-text-primary focus:outline-none focus:border-primary-500"
                  placeholder="0.0.0.0"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-text-secondary mb-1">
                  Port
                </label>
                <input
                  type="number"
                  required
                  min="1"
                  max="65535"
                  value={formData.port}
                  onChange={(e) =>
                    setFormData({ ...formData, port: parseInt(e.target.value) })
                  }
                  className="w-full px-3 py-2 bg-dark-bg-tertiary border border-gray-700 rounded text-dark-text-primary focus:outline-none focus:border-primary-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-text-secondary mb-1">
                  URI Path
                </label>
                <input
                  type="text"
                  required
                  value={formData.uriPath}
                  onChange={(e) =>
                    setFormData({ ...formData, uriPath: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-dark-bg-tertiary border border-gray-700 rounded text-dark-text-primary focus:outline-none focus:border-primary-500"
                  placeholder="api/beacon"
                />
                <p className="mt-1 text-xs text-dark-text-secondary">
                  Path where the listener will receive callbacks (without leading /)
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-text-secondary mb-1">
                  Encryption Key
                </label>
                <div className="flex gap-2">
                  <input
                    type="password"
                    required
                    minLength={32}
                    value={formData.encryptionKey}
                    onChange={(e) =>
                      setFormData({ ...formData, encryptionKey: e.target.value })
                    }
                    className="flex-1 px-3 py-2 bg-dark-bg-tertiary border border-gray-700 rounded text-dark-text-primary focus:outline-none focus:border-primary-500"
                    placeholder="At least 32 characters"
                  />
                  <button
                    type="button"
                    onClick={generateEncryptionKey}
                    className="px-3 py-2 bg-primary-500 hover:bg-primary-600 text-white rounded transition-colors whitespace-nowrap"
                  >
                    Generate
                  </button>
                </div>
                <p className="mt-1 text-xs text-dark-text-secondary">
                  AES-256 encryption key (minimum 32 characters)
                </p>
              </div>

              {formData.type === 'https' && (
                <>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="tlsEnabled"
                      checked={formData.tlsEnabled}
                      onChange={(e) =>
                        setFormData({ ...formData, tlsEnabled: e.target.checked })
                      }
                      className="w-4 h-4"
                    />
                    <label
                      htmlFor="tlsEnabled"
                      className="text-sm text-dark-text-secondary"
                    >
                      Enable TLS
                    </label>
                  </div>

                  {formData.tlsEnabled && (
                    <>
                      <div>
                        <label className="block text-sm font-medium text-dark-text-secondary mb-1">
                          Certificate File
                        </label>
                        <input
                          type="text"
                          value={formData.certFile || ''}
                          onChange={(e) =>
                            setFormData({ ...formData, certFile: e.target.value })
                          }
                          className="w-full px-3 py-2 bg-dark-bg-tertiary border border-gray-700 rounded text-dark-text-primary focus:outline-none focus:border-primary-500"
                          placeholder="/path/to/cert.pem"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-dark-text-secondary mb-1">
                          Key File
                        </label>
                        <input
                          type="text"
                          value={formData.keyFile || ''}
                          onChange={(e) =>
                            setFormData({ ...formData, keyFile: e.target.value })
                          }
                          className="w-full px-3 py-2 bg-dark-bg-tertiary border border-gray-700 rounded text-dark-text-primary focus:outline-none focus:border-primary-500"
                          placeholder="/path/to/key.pem"
                        />
                      </div>
                    </>
                  )}
                </>
              )}

              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowCreateForm(false)}
                  className="flex-1 px-4 py-2 bg-dark-bg-tertiary hover:bg-gray-700 text-dark-text-primary rounded transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="flex-1 px-4 py-2 bg-primary-500 hover:bg-primary-600 text-white rounded transition-colors"
                >
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
