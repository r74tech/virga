import { useEffect, useState } from 'react';
import { RefreshCw } from 'lucide-react';
import type { Session } from '../../types/session';
import { apiClient } from '../../services/apiClient';
import { SessionsApi } from '../../services/api/sessions';

interface SessionSelectorProps {
  selectedSessionId: string | null;
  onSessionSelect: (session: Session) => void;
}

export function SessionSelector({
  selectedSessionId,
  onSessionSelect,
}: SessionSelectorProps) {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sessionsApi = new SessionsApi(apiClient);

  useEffect(() => {
    loadSessions();
  }, []);

  const loadSessions = async () => {
    setLoading(true);
    setError(null);

    try {
      const data = await sessionsApi.getSessions();
      setSessions(data);

      // Auto-select: First active session
      if (data.length > 0 && !selectedSessionId) {
        const activeSession = data.find((s) => s.status === 'active');
        if (activeSession) {
          onSessionSelect(activeSession);
        } else if (data[0]) {
          // If no active session, select the first session
          onSessionSelect(data[0]);
        }
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load sessions';
      setError(errorMessage);
      console.error('Failed to load sessions:', err);
    } finally {
      setLoading(false);
    }
  };

  const selectedSession = sessions.find((s) => s.id === selectedSessionId);

  const handleSessionChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const session = sessions.find((s) => s.id === event.target.value);
    if (session) {
      onSessionSelect(session);
    }
  };

  return (
    <div className="flex items-center gap-3">
      <span className="text-sm text-dark-text-secondary">Session:</span>

      {loading ? (
        <div className="flex items-center gap-2 text-sm text-dark-text-secondary">
          <RefreshCw className="w-4 h-4 animate-spin" />
          <span>Loading...</span>
        </div>
      ) : error ? (
        <div className="text-sm text-red-400">{error}</div>
      ) : (
        <>
          <select
            value={selectedSessionId || ''}
            onChange={handleSessionChange}
            className="px-3 py-1.5 rounded-lg bg-dark-bg-tertiary border border-gray-700 text-dark-text-primary focus:outline-none focus:border-primary-500 transition-colors"
          >
            <option value="">Select a session...</option>
            {sessions.map((session) => (
              <option key={session.id} value={session.id}>
                {session.hostname} ({session.username}@{session.os})
                {session.status === 'active' ? ' ●' : ' ○'}
              </option>
            ))}
          </select>

          <button
            onClick={loadSessions}
            disabled={loading}
            className="p-1.5 rounded-lg bg-dark-bg-tertiary hover:bg-dark-bg-secondary border border-gray-700 text-dark-text-secondary hover:text-dark-text-primary transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Refresh sessions"
            type="button"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>

          {selectedSession && (
            <div className="flex items-center gap-3 text-sm text-dark-text-secondary">
              <span>IP: {selectedSession.ip}</span>
              <span className="hidden sm:inline">
                Last activity: {new Date(selectedSession.lastActivity).toLocaleString()}
              </span>
            </div>
          )}
        </>
      )}
    </div>
  );
}
