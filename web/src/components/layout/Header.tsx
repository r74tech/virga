import { Activity, Terminal } from 'lucide-react';
import { useSessionStore } from '../../stores/sessionStore';

export const Header = () => {
  const stats = useSessionStore((state) => state.stats);

  return (
    <header className="h-16 bg-dark-bg-secondary border-b border-primary-900/30 flex items-center px-6">
      <div className="flex items-center gap-3">
        <Terminal className="w-8 h-8 text-primary-500" />
        <h1 className="text-2xl font-bold text-dark-text-primary">Virga C2</h1>
      </div>

      <div className="ml-auto flex items-center gap-6">
        <div className="flex items-center gap-2">
          <Activity className="w-5 h-5 text-primary-500" />
          <span className="text-sm text-dark-text-secondary">
            {stats.activeSessions} / {stats.totalSessions} Active
          </span>
        </div>

        <div className="flex items-center gap-2 text-xs text-dark-text-secondary">
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-green-500"></div>
            <span>{stats.activeSessions}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-yellow-500"></div>
            <span>{stats.inactiveSessions}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-red-500"></div>
            <span>{stats.disconnectedSessions}</span>
          </div>
        </div>
      </div>
    </header>
  );
};
