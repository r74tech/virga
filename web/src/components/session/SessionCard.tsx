import { Monitor, User, Clock, Activity } from 'lucide-react';
import type { Session } from '../../types/session';
import { formatDistanceToNow } from 'date-fns';

interface SessionCardProps {
  session: Session;
  isSelected?: boolean;
  onClick?: () => void;
}

export const SessionCard = ({
  session,
  isSelected = false,
  onClick,
}: SessionCardProps) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-500';
      case 'inactive':
        return 'bg-red-500';
      case 'disconnected':
        return 'bg-gray-500';
      default:
        return 'bg-gray-500';
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
      return 'text-red-400';
    }
    return 'text-dark-text-secondary';
  };

  return (
    <div
      onClick={onClick}
      className={`p-4 rounded-lg border cursor-pointer transition-all ${
        isSelected
          ? 'border-primary-500 bg-dark-bg-tertiary'
          : 'border-gray-700 bg-dark-bg-secondary hover:border-gray-600'
      }`}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <Monitor className="w-5 h-5 text-primary-500" />
          <h3 className="font-semibold text-dark-text-primary">
            {session.hostname}
          </h3>
        </div>
        <div className="flex items-center gap-2">
          <div className={`w-2 h-2 rounded-full ${getStatusColor(session.status)}`} />
          <span className="text-xs text-dark-text-secondary capitalize">
            {session.status}
          </span>
        </div>
      </div>

      <div className="space-y-2 text-sm">
        <div className="flex items-center gap-2 text-dark-text-secondary">
          <User className="w-4 h-4" />
          <span>{session.username}</span>
          {session.privilege && (
            <span className={`ml-1 ${getPrivilegeColor(session.privilege)}`}>
              ({session.privilege})
            </span>
          )}
        </div>

        <div className="flex items-center gap-2 text-dark-text-secondary">
          <Activity className="w-4 h-4" />
          <span>{session.ip}</span>
        </div>

        <div className="flex items-center gap-2 text-dark-text-secondary">
          <Monitor className="w-4 h-4" />
          <span>{session.os}</span>
        </div>

        <div className="flex items-center gap-2 text-dark-text-secondary">
          <Clock className="w-4 h-4" />
          <span className="text-xs">
            {formatDistanceToNow(session.lastActivity, { addSuffix: true })}
          </span>
        </div>
      </div>

      {session.metadata?.department && (
        <div className="mt-3 pt-3 border-t border-gray-700">
          <span className="text-xs text-dark-text-secondary">
            {session.metadata.department}
          </span>
        </div>
      )}
    </div>
  );
};
