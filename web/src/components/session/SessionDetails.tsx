import { Monitor, User, Activity, Clock, HardDrive, Cpu, Link2, Settings } from 'lucide-react';
import type { Session } from '../../types/session';
import { format, formatDistanceToNow } from 'date-fns';
import { useState, useEffect } from 'react';
import { apiClient } from '../../services/apiClient';

interface SessionDetailsProps {
  session: Session;
}

export const SessionDetails = ({ session }: SessionDetailsProps) => {
  const [beaconInterval, setBeaconInterval] = useState(session.interval ?? 30);
  const [beaconJitter, setBeaconJitter] = useState(session.jitter ?? 10);
  const [isUpdating, setIsUpdating] = useState(false);

  // Update state when session changes
  useEffect(() => {
    setBeaconInterval(session.interval ?? 30);
    setBeaconJitter(session.jitter ?? 10);
  }, [session.interval, session.jitter]);

  const handleUpdateBeaconConfig = async () => {
    setIsUpdating(true);
    try {
      const response = await apiClient.post<{
        success: boolean;
        message: string;
        config?: { sleep_time: number; jitter: number };
      }>(`/api/sessions/${session.id}/beacon-config`, {
        sleep_time: beaconInterval,
        jitter: beaconJitter,
      });

      if (response.success) {
        alert(`Beacon configuration updated successfully!\nInterval: ${beaconInterval}s, Jitter: ${beaconJitter}%`);
      } else {
        alert(`Failed to update beacon config: ${response.message}`);
      }
    } catch (error) {
      console.error('Failed to update beacon config:', error);
      const errorMsg = error instanceof Error ? error.message : 'Unknown error occurred';
      alert(`Failed to update beacon config: ${errorMsg}`);
    } finally {
      setIsUpdating(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'text-green-500';
      case 'inactive':
        return 'text-red-500';
      case 'disconnected':
        return 'text-gray-500';
      default:
        return 'text-gray-500';
    }
  };

  const getPrivilegeColor = (privilege?: string) => {
    if (!privilege) return 'text-dark-text-secondary';

    const lower = privilege.toLowerCase();
    if (
      lower.includes('admin') ||
      lower.includes('system') ||
      lower.includes('root')
    ) {
      return 'text-red-400 font-semibold';
    }
    return 'text-dark-text-secondary';
  };

  return (
    <div className="h-full overflow-y-auto bg-dark-bg p-6">
      <h3 className="text-lg font-semibold text-dark-text-primary mb-4">
        Session Details
      </h3>

      <div className="space-y-6">
        {/* Basic Info */}
        <div className="space-y-3">
          <div className="flex items-start gap-3">
            <Monitor className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="text-xs text-dark-text-secondary mb-1">Hostname</p>
              <p className="text-sm text-dark-text-primary font-mono break-all">
                {session.hostname}
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3">
            <User className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="text-xs text-dark-text-secondary mb-1">User</p>
              <p className="text-sm text-dark-text-primary font-mono break-all">
                {session.username}
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3">
            <Activity className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-xs text-dark-text-secondary mb-1">Status</p>
              <p className={`text-sm font-medium ${getStatusColor(session.status)}`}>
                {session.status.charAt(0).toUpperCase() + session.status.slice(1)}
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3">
            <Clock className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-xs text-dark-text-secondary mb-1">Last Activity</p>
              <p className="text-sm text-dark-text-primary">
                {formatDistanceToNow(session.lastActivity, { addSuffix: true })}
              </p>
              <p className="text-xs text-dark-text-secondary mt-1">
                {format(session.lastActivity, 'PPpp')}
              </p>
            </div>
          </div>
        </div>

        {/* System Info */}
        <div className="pt-4 border-t border-gray-700">
          <h4 className="text-sm font-semibold text-dark-text-primary mb-3">
            System Information
          </h4>
          <div className="space-y-3">
            <div className="flex items-start gap-3">
              <HardDrive className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
              <div className="flex-1">
                <p className="text-xs text-dark-text-secondary mb-1">
                  Operating System
                </p>
                <p className="text-sm text-dark-text-primary">{session.os}</p>
              </div>
            </div>

            <div className="flex items-start gap-3">
              <Cpu className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
              <div className="flex-1">
                <p className="text-xs text-dark-text-secondary mb-1">
                  Architecture
                </p>
                <p className="text-sm text-dark-text-primary">{session.arch}</p>
              </div>
            </div>

            {session.privilege && (
              <div className="flex items-start gap-3">
                <User className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
                <div className="flex-1">
                  <p className="text-xs text-dark-text-secondary mb-1">Privilege</p>
                  <p className={`text-sm ${getPrivilegeColor(session.privilege)}`}>
                    {session.privilege}
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Network Info */}
        <div className="pt-4 border-t border-gray-700">
          <h4 className="text-sm font-semibold text-dark-text-primary mb-3">
            Network Information
          </h4>
          <div className="space-y-3">
            <div className="flex items-start gap-3">
              <Link2 className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
              <div className="flex-1 min-w-0">
                <p className="text-xs text-dark-text-secondary mb-1">IP Address</p>
                <p className="text-sm text-dark-text-primary font-mono break-all">
                  {session.ip}
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3">
              <Link2 className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
              <div className="flex-1">
                <p className="text-xs text-dark-text-secondary mb-1">
                  Connection Type
                </p>
                <p className="text-sm text-dark-text-primary">
                  {session.connectionType === 'direct' ? 'Direct C2' : 'Pivot'}
                </p>
                {session.pivotChain && session.pivotChain.length > 0 && (
                  <p className="text-xs text-dark-text-secondary mt-1">
                    Via: {session.pivotChain.join(' → ')}
                  </p>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Metadata */}
        {session.metadata && Object.keys(session.metadata).length > 0 && (
          <div className="pt-4 border-t border-gray-700">
            <h4 className="text-sm font-semibold text-dark-text-primary mb-3">
              Additional Information
            </h4>
            <div className="space-y-2">
              {Object.entries(session.metadata).map(([key, value]) => (
                <div key={key} className="flex justify-between items-start gap-4">
                  <span className="text-xs text-dark-text-secondary">
                    {key.charAt(0).toUpperCase() +
                      key.slice(1).replace(/([A-Z])/g, ' $1')}
                    :
                  </span>
                  <span className="text-xs text-dark-text-primary text-right break-all">
                    {String(value)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Beacon Configuration */}
        <div className="pt-4 border-t border-gray-700">
          <div className="flex items-center gap-2 mb-3">
            <Settings className="w-4 h-4 text-primary-500" />
            <h4 className="text-sm font-semibold text-dark-text-primary">
              Beacon Configuration
            </h4>
          </div>
          <div className="space-y-4">
            <div>
              <label className="block text-xs text-dark-text-secondary mb-2">
                Beacon Interval (seconds)
              </label>
              <input
                type="number"
                min="1"
                max="300"
                value={beaconInterval}
                onChange={(e) => setBeaconInterval(Number(e.target.value))}
                className="w-full px-3 py-2 bg-dark-bg-tertiary border border-gray-600 rounded text-sm text-dark-text-primary focus:outline-none focus:border-primary-500"
              />
              <p className="text-xs text-dark-text-secondary mt-1">
                How often the beacon checks in (1-300 seconds)
              </p>
            </div>

            <div>
              <label className="block text-xs text-dark-text-secondary mb-2">
                Jitter (%)
              </label>
              <input
                type="number"
                min="0"
                max="50"
                value={beaconJitter}
                onChange={(e) => setBeaconJitter(Number(e.target.value))}
                className="w-full px-3 py-2 bg-dark-bg-tertiary border border-gray-600 rounded text-sm text-dark-text-primary focus:outline-none focus:border-primary-500"
              />
              <p className="text-xs text-dark-text-secondary mt-1">
                Randomization of interval (0-50%)
              </p>
            </div>

            <button
              onClick={handleUpdateBeaconConfig}
              disabled={isUpdating}
              className="w-full px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-gray-600 text-white text-sm font-medium rounded transition-colors"
            >
              {isUpdating ? 'Updating...' : 'Update Configuration'}
            </button>

            <div className="text-xs text-dark-text-secondary p-3 bg-dark-bg-tertiary rounded">
              <p className="font-medium mb-1">Current: {beaconInterval}s ± {beaconJitter}%</p>
              <p>Actual range: {Math.floor(beaconInterval * (1 - beaconJitter / 100))}s - {Math.ceil(beaconInterval * (1 + beaconJitter / 100))}s</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
