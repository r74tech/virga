import { useState } from 'react';
import { Search, Filter, AlertCircle, Loader2 } from 'lucide-react';
import { SessionCard } from '../components/session/SessionCard';
import { SessionDetails } from '../components/session/SessionDetails';
import { SessionTerminal } from '../components/session/SessionTerminal';
import { useSessionStore } from '../stores/sessionStore';
import { useAutoRefresh } from '../hooks/useAutoRefresh';
import type { Session } from '../types/session';

export const SessionsPage = () => {
  const sessions = useSessionStore((state) => state.sessions);
  const selectedSession = useSessionStore((state) => state.selectedSession);
  const setSelectedSession = useSessionStore(
    (state) => state.setSelectedSession
  );
  const stats = useSessionStore((state) => state.stats);
  const isLoading = useSessionStore((state) => state.isLoading);
  const error = useSessionStore((state) => state.error);
  const fetchSessions = useSessionStore((state) => state.fetchSessions);
  const clearError = useSessionStore((state) => state.clearError);

  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>(
    'all'
  );

  // Auto-refresh sessions every 5 seconds
  useAutoRefresh(fetchSessions, {
    enabled: true,
    interval: 5000,
  });

  const filteredSessions = sessions.filter((session) => {
    const matchesSearch =
      session.hostname.toLowerCase().includes(searchQuery.toLowerCase()) ||
      session.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
      session.ip.includes(searchQuery);

    const matchesStatus =
      statusFilter === 'all' || session.status === statusFilter;

    return matchesSearch && matchesStatus;
  });

  const handleSessionClick = (session: Session) => {
    setSelectedSession(session);
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-gray-700 flex-shrink-0">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-2xl font-bold text-dark-text-primary">
            Sessions
          </h1>
          <div className="flex gap-4 text-sm">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-green-500" />
              <span className="text-dark-text-secondary">
                Active: {stats.activeSessions}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-yellow-500" />
              <span className="text-dark-text-secondary">
                Inactive: {stats.inactiveSessions}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-red-500" />
              <span className="text-dark-text-secondary">
                Disconnected: {stats.disconnectedSessions}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-dark-text-secondary">
                Total: {stats.totalSessions}
              </span>
            </div>
          </div>
        </div>

        {/* Search and Filter */}
        <div className="flex gap-4">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-dark-text-secondary" />
            <input
              type="text"
              placeholder="Search by hostname, username, or IP..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-dark-bg-secondary border border-gray-700 rounded-lg text-dark-text-primary placeholder-dark-text-secondary focus:outline-none focus:border-primary-500"
            />
          </div>

          <div className="flex items-center gap-2 px-4 py-2 bg-dark-bg-secondary border border-gray-700 rounded-lg">
            <Filter className="w-5 h-5 text-dark-text-secondary" />
            <select
              value={statusFilter}
              onChange={(e) =>
                setStatusFilter(e.target.value as 'all' | 'active' | 'inactive')
              }
              className="bg-transparent text-dark-text-primary focus:outline-none cursor-pointer"
            >
              <option value="all">All Status</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
          </div>
        </div>
      </div>

      {/* Main Content Area - 3 Pane Layout */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left Pane: Session List */}
        <div className="w-80 border-r border-gray-700 overflow-y-auto p-4 flex-shrink-0">
          {/* Error Display */}
          {error && (
            <div className="mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg">
              <div className="flex items-start gap-3">
                <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
                <div className="flex-1">
                  <p className="text-red-400 text-sm font-medium mb-1">
                    Failed to load sessions
                  </p>
                  <p className="text-red-300 text-xs">{error}</p>
                  <button
                    type="button"
                    onClick={() => {
                      clearError();
                      fetchSessions();
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
          {isLoading && sessions.length === 0 ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <Loader2 className="w-8 h-8 text-primary-500 animate-spin mx-auto mb-2" />
                <p className="text-dark-text-secondary">Loading sessions...</p>
              </div>
            </div>
          ) : filteredSessions.length === 0 ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <p className="text-dark-text-secondary">No sessions found</p>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              {filteredSessions.map((session) => (
                <SessionCard
                  key={session.id}
                  session={session}
                  isSelected={selectedSession?.id === session.id}
                  onClick={() => handleSessionClick(session)}
                />
              ))}
            </div>
          )}
        </div>

        {/* Right Side: Details (top) + Terminal (bottom) */}
        {selectedSession ? (
          <div className="flex-1 flex flex-col overflow-hidden">
            {/* Top: Session Details */}
            <div className="h-80 border-b border-gray-700 overflow-y-auto flex-shrink-0">
              <SessionDetails session={selectedSession} />
            </div>

            {/* Bottom: Terminal */}
            <div className="flex-1 overflow-hidden">
              <SessionTerminal session={selectedSession} />
            </div>
          </div>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <p className="text-dark-text-secondary text-lg">
                Select a session to view details and open terminal
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
