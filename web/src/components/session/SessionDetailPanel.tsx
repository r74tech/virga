import { X, Monitor, User, Activity, Clock, Cpu, Link2, Terminal, AlertTriangle } from 'lucide-react';
import { useState } from 'react';
import type { Session } from '../../types/session';
import { format, formatDistanceToNow } from 'date-fns';
import { SessionTerminal } from './SessionTerminal';
import { api } from '../../services/api';

interface SessionDetailPanelProps {
  session: Session;
  onClose: () => void;
}

export const SessionDetailPanel = ({
  session,
  onClose,
}: SessionDetailPanelProps) => {
  const [showTerminal, setShowTerminal] = useState(false);
  const [showKillswitchDialog, setShowKillswitchDialog] = useState(false);
  const [killswitchMode, setKillswitchMode] = useState<'shutdown' | 'remove'>('shutdown');
  const [killswitchMessage, setKillswitchMessage] = useState('');
  const [killswitchConfirmation, setKillswitchConfirmation] = useState('');
  const [isKillswitching, setIsKillswitching] = useState(false);
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

  const handleKillswitch = async () => {
    if (killswitchConfirmation !== 'YES') {
      alert('You must type "YES" to confirm killswitch activation');
      return;
    }

    setIsKillswitching(true);
    try {
      await api.sessions.killswitch(session.id, {
        mode: killswitchMode,
        message: killswitchMessage,
      });

      alert(`Killswitch activated successfully!\nMode: ${killswitchMode}\nThe implant will terminate shortly.`);
      setShowKillswitchDialog(false);
      onClose();
    } catch (error) {
      console.error('Failed to activate killswitch:', error);
      alert(`Failed to activate killswitch: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setIsKillswitching(false);
    }
  };

  if (showTerminal) {
    return (
      <div className="absolute top-4 right-4 w-[800px] h-[600px] bg-dark-bg-tertiary border border-gray-700 rounded-lg shadow-xl overflow-hidden flex flex-col">
        <div className="flex items-center justify-between p-3 border-b border-gray-700 bg-dark-bg-secondary">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setShowTerminal(false)}
              className="px-3 py-1 text-sm bg-dark-bg-tertiary hover:bg-dark-bg rounded transition-colors"
            >
              ← Back
            </button>
            <h3 className="text-sm font-semibold text-dark-text-primary">
              Terminal - {session.hostname}
            </h3>
          </div>
          <button
            onClick={onClose}
            className="p-1 hover:bg-dark-bg-tertiary rounded transition-colors"
          >
            <X className="w-5 h-5 text-dark-text-secondary" />
          </button>
        </div>
        <div className="flex-1 overflow-hidden">
          <SessionTerminal session={session} />
        </div>
      </div>
    );
  }

  return (
    <div className="absolute top-4 right-4 w-96 bg-dark-bg-tertiary border border-gray-700 rounded-lg shadow-xl overflow-hidden">
      <div className="flex items-center justify-between p-4 border-b border-gray-700">
        <h3 className="text-lg font-semibold text-dark-text-primary">
          Session Details
        </h3>
        <button
          onClick={onClose}
          className="p-1 hover:bg-dark-bg-secondary rounded transition-colors"
        >
          <X className="w-5 h-5 text-dark-text-secondary" />
        </button>
      </div>

      <div className="p-4 space-y-4 max-h-[700px] overflow-y-auto">
        <div>
          <label className="text-sm text-dark-text-secondary">Hostname</label>
          <p className="text-dark-text-primary font-semibold text-lg flex items-center gap-2">
            <Monitor className="w-5 h-5 text-primary-500" />
            {session.hostname}
          </p>
        </div>

        <div>
          <label className="text-sm text-dark-text-secondary">Status</label>
          <p className={`font-medium ${getStatusColor(session.status)}`}>
            {session.status.toUpperCase()}
          </p>
        </div>

        <div>
          <label className="text-sm text-dark-text-secondary flex items-center gap-2">
            <Link2 className="w-4 h-4" />
            Connection Type
          </label>
          <div className="flex items-center gap-2 mt-1">
            <span
              className={`px-2 py-1 text-xs rounded ${
                session.connectionType === 'direct'
                  ? 'bg-green-500/20 text-green-400'
                  : 'bg-yellow-500/20 text-yellow-400'
              }`}
            >
              {session.connectionType === 'direct' ? 'Direct C2' : 'Pivot'}
            </span>
          </div>
          {session.pivotChain && session.pivotChain.length > 0 && (
            <div className="mt-2 p-2 bg-dark-bg-secondary rounded">
              <p className="text-xs text-dark-text-secondary mb-1">Pivot Chain:</p>
              <div className="flex flex-col gap-1">
                {session.pivotChain.map((hop, index) => (
                  <div key={index} className="flex items-center gap-2 text-xs">
                    <span className="text-dark-text-secondary">{index + 1}.</span>
                    <span className="text-dark-text-primary font-mono">{hop}</span>
                  </div>
                ))}
                <div className="flex items-center gap-2 text-xs">
                  <span className="text-dark-text-secondary">→</span>
                  <span className="text-primary-400 font-mono">{session.hostname}</span>
                </div>
              </div>
            </div>
          )}
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm text-dark-text-secondary">Username</label>
            <p className="text-dark-text-primary font-mono text-sm flex items-center gap-2">
              <User className="w-4 h-4" />
              {session.username}
            </p>
          </div>

          <div>
            <label className="text-sm text-dark-text-secondary">Privilege</label>
            <p className={`font-mono text-sm ${getPrivilegeColor(session.privilege)}`}>
              {session.privilege || 'N/A'}
            </p>
          </div>
        </div>

        <div>
          <label className="text-sm text-dark-text-secondary">IP Address</label>
          <p className="text-dark-text-primary font-mono text-sm flex items-center gap-2">
            <Activity className="w-4 h-4" />
            {session.ip}
          </p>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm text-dark-text-secondary">OS</label>
            <p className="text-dark-text-primary text-sm">{session.os}</p>
          </div>

          <div>
            <label className="text-sm text-dark-text-secondary">Architecture</label>
            <p className="text-dark-text-primary text-sm flex items-center gap-2">
              <Cpu className="w-4 h-4" />
              {session.arch}
            </p>
          </div>
        </div>

        {session.processId && (
          <div>
            <label className="text-sm text-dark-text-secondary">Process ID</label>
            <p className="text-dark-text-primary font-mono text-sm">
              {session.processId}
            </p>
          </div>
        )}

        <div>
          <label className="text-sm text-dark-text-secondary">Last Activity</label>
          <p className="text-dark-text-primary text-sm flex items-center gap-2">
            <Clock className="w-4 h-4" />
            {format(session.lastActivity, 'yyyy-MM-dd HH:mm:ss')}
          </p>
          <p className="text-dark-text-secondary text-xs mt-1">
            ({formatDistanceToNow(session.lastActivity, { addSuffix: true })})
          </p>
        </div>

        {session.metadata && (
          <>
            {session.metadata.beaconInterval && (
              <div>
                <label className="text-sm text-dark-text-secondary">
                  Beacon Interval
                </label>
                <p className="text-dark-text-primary text-sm">
                  {session.metadata.beaconInterval}s
                  {session.metadata.jitter && ` (±${session.metadata.jitter}s jitter)`}
                </p>
              </div>
            )}

            {session.metadata.lastCheckin && (
              <div>
                <label className="text-sm text-dark-text-secondary">
                  Last Checkin
                </label>
                <p className="text-dark-text-primary text-sm">
                  {format(
                    new Date(session.metadata.lastCheckin),
                    'yyyy-MM-dd HH:mm:ss'
                  )}
                </p>
              </div>
            )}

            {session.metadata.missedCheckins !== undefined && (
              <div>
                <label className="text-sm text-dark-text-secondary">
                  Missed Checkins
                </label>
                <p
                  className={`text-sm font-medium ${
                    session.metadata.missedCheckins > 2
                      ? 'text-red-400'
                      : 'text-dark-text-primary'
                  }`}
                >
                  {session.metadata.missedCheckins}
                </p>
              </div>
            )}

            {session.metadata.domain && (
              <div>
                <label className="text-sm text-dark-text-secondary">Domain</label>
                <p className="text-dark-text-primary text-sm">
                  {session.metadata.domain}
                </p>
              </div>
            )}

            {session.metadata.department && (
              <div>
                <label className="text-sm text-dark-text-secondary">
                  Department
                </label>
                <p className="text-dark-text-primary text-sm">
                  {session.metadata.department}
                </p>
              </div>
            )}

            {session.metadata.role && (
              <div>
                <label className="text-sm text-dark-text-secondary">Role</label>
                <p className="text-dark-text-primary text-sm">
                  {session.metadata.role}
                </p>
              </div>
            )}
          </>
        )}

        <div className="pt-4 border-t border-gray-700">
          <label className="text-sm text-dark-text-secondary mb-2 block">
            Session IDs
          </label>
          <div className="bg-dark-bg-secondary p-3 rounded space-y-2">
            <div>
              <span className="text-xs text-dark-text-secondary">Session ID:</span>
              <p className="text-dark-text-primary font-mono text-xs break-all">
                {session.id}
              </p>
            </div>
            <div>
              <span className="text-xs text-dark-text-secondary">Agent ID:</span>
              <p className="text-dark-text-primary font-mono text-xs break-all">
                {session.agentId}
              </p>
            </div>
          </div>
        </div>

        {session.status === 'active' && (
          <div className="pt-4 border-t border-gray-700 space-y-2">
            <button
              onClick={() => setShowTerminal(true)}
              className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 rounded-lg transition-colors"
            >
              <Terminal className="w-4 h-4" />
              <span className="font-medium">Open Terminal</span>
            </button>

            <button
              onClick={() => setShowKillswitchDialog(true)}
              className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg transition-colors"
            >
              <AlertTriangle className="w-4 h-4" />
              <span className="font-medium">Killswitch</span>
            </button>
          </div>
        )}
      </div>

      {/* Killswitch Confirmation Dialog */}
      {showKillswitchDialog && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-dark-bg-tertiary border border-red-500/50 rounded-lg p-6 max-w-md w-full mx-4">
            <div className="flex items-center gap-3 mb-4">
              <AlertTriangle className="w-6 h-6 text-red-500" />
              <h3 className="text-lg font-semibold text-dark-text-primary">
                KILLSWITCH WARNING
              </h3>
            </div>

            <div className="space-y-4 mb-6">
              <div className="bg-red-500/10 border border-red-500/30 rounded p-3">
                <p className="text-sm text-dark-text-primary">
                  You are about to activate the killswitch for:
                </p>
                <p className="text-sm font-semibold text-dark-text-primary mt-2">
                  {session.hostname} ({session.username})
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-text-secondary mb-2">
                  Mode:
                </label>
                <div className="space-y-2">
                  <label className="flex items-start gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="killswitch-mode"
                      value="shutdown"
                      checked={killswitchMode === 'shutdown'}
                      onChange={(e) => setKillswitchMode(e.target.value as 'shutdown' | 'remove')}
                      className="mt-1"
                    />
                    <div>
                      <div className="text-sm font-medium text-dark-text-primary">Shutdown</div>
                      <div className="text-xs text-dark-text-secondary">
                        Stop the implant cleanly. The binary file remains on the target system.
                      </div>
                    </div>
                  </label>
                  <label className="flex items-start gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="killswitch-mode"
                      value="remove"
                      checked={killswitchMode === 'remove'}
                      onChange={(e) => setKillswitchMode(e.target.value as 'shutdown' | 'remove')}
                      className="mt-1"
                    />
                    <div>
                      <div className="text-sm font-medium text-red-400">Remove</div>
                      <div className="text-xs text-dark-text-secondary">
                        Delete the implant binary and stop. This action is PERMANENT and cannot be undone.
                      </div>
                    </div>
                  </label>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-text-secondary mb-2">
                  Message (optional):
                </label>
                <input
                  type="text"
                  value={killswitchMessage}
                  onChange={(e) => setKillswitchMessage(e.target.value)}
                  placeholder="Reason for killswitch activation"
                  className="w-full px-3 py-2 bg-dark-bg-secondary border border-gray-700 rounded text-dark-text-primary text-sm"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-text-secondary mb-2">
                  Type <span className="text-red-500 font-bold">YES</span> to confirm:
                </label>
                <input
                  type="text"
                  value={killswitchConfirmation}
                  onChange={(e) => setKillswitchConfirmation(e.target.value)}
                  placeholder="YES"
                  className="w-full px-3 py-2 bg-dark-bg-secondary border border-gray-700 rounded text-dark-text-primary text-sm font-mono"
                />
              </div>
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => {
                  setShowKillswitchDialog(false);
                  setKillswitchConfirmation('');
                  setKillswitchMessage('');
                  setKillswitchMode('shutdown');
                }}
                disabled={isKillswitching}
                className="flex-1 px-4 py-2 bg-dark-bg-secondary hover:bg-dark-bg rounded-lg transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleKillswitch}
                disabled={isKillswitching || killswitchConfirmation !== 'YES'}
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg transition-colors font-semibold disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isKillswitching ? 'Activating...' : 'Activate Killswitch'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
