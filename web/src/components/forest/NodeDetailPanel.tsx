import { X } from 'lucide-react';
import type { Node } from '../../types/graph';
import { format } from 'date-fns';

interface NodeDetailPanelProps {
  node: Node;
  onClose: () => void;
}

export const NodeDetailPanel = ({ node, onClose }: NodeDetailPanelProps) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'text-green-500';
      case 'inactive':
        return 'text-red-500';
      case 'disconnected':
        return 'text-gray-500';
      default:
        return 'text-gray-400';
    }
  };

  const getTypeLabel = (type: string) => {
    switch (type) {
      case 'system':
        return 'C2 Server';
      case 'workstation':
        return 'Workstation';
      case 'server':
        return 'Server';
      case 'network':
        return 'Network';
      default:
        return type;
    }
  };

  return (
    <div className="absolute top-4 right-4 w-80 bg-dark-bg-tertiary border border-gray-700 rounded-lg shadow-xl overflow-hidden">
      <div className="flex items-center justify-between p-4 border-b border-gray-700">
        <h3 className="text-lg font-semibold text-dark-text-primary">
          Node Details
        </h3>
        <button
          onClick={onClose}
          className="p-1 hover:bg-dark-bg-secondary rounded transition-colors"
        >
          <X className="w-5 h-5 text-dark-text-secondary" />
        </button>
      </div>

      <div className="p-4 space-y-4 max-h-[600px] overflow-y-auto">
        <div>
          <label className="text-sm text-dark-text-secondary">Label</label>
          <p className="text-dark-text-primary font-medium">{node.label}</p>
        </div>

        <div>
          <label className="text-sm text-dark-text-secondary">Type</label>
          <p className="text-dark-text-primary">{getTypeLabel(node.type)}</p>
        </div>

        <div>
          <label className="text-sm text-dark-text-secondary">Status</label>
          <p className={`font-medium ${getStatusColor(node.status)}`}>
            {node.status.toUpperCase()}
          </p>
        </div>

        {node.properties.hostname && (
          <div>
            <label className="text-sm text-dark-text-secondary">Hostname</label>
            <p className="text-dark-text-primary font-mono text-sm">
              {node.properties.hostname}
            </p>
          </div>
        )}

        {node.properties.ip && (
          <div>
            <label className="text-sm text-dark-text-secondary">IP Address</label>
            <p className="text-dark-text-primary font-mono text-sm">
              {node.properties.ip}
            </p>
          </div>
        )}

        {node.properties.os && (
          <div>
            <label className="text-sm text-dark-text-secondary">
              Operating System
            </label>
            <p className="text-dark-text-primary">{node.properties.os}</p>
          </div>
        )}

        {node.properties.username && (
          <div>
            <label className="text-sm text-dark-text-secondary">Username</label>
            <p className="text-dark-text-primary font-mono text-sm">
              {node.properties.username}
            </p>
          </div>
        )}

        {node.properties.privilege && (
          <div>
            <label className="text-sm text-dark-text-secondary">Privilege</label>
            <p className="text-dark-text-primary">{node.properties.privilege}</p>
          </div>
        )}

        <div>
          <label className="text-sm text-dark-text-secondary">Created At</label>
          <p className="text-dark-text-primary text-sm">
            {format(node.createdAt, 'yyyy-MM-dd HH:mm:ss')}
          </p>
        </div>

        <div>
          <label className="text-sm text-dark-text-secondary">Updated At</label>
          <p className="text-dark-text-primary text-sm">
            {format(node.updatedAt, 'yyyy-MM-dd HH:mm:ss')}
          </p>
        </div>

        {Object.keys(node.properties).length > 0 && (
          <div>
            <label className="text-sm text-dark-text-secondary mb-2 block">
              Additional Properties
            </label>
            <div className="bg-dark-bg-secondary p-3 rounded space-y-2">
              {Object.entries(node.properties)
                .filter(
                  ([key]) =>
                    !['hostname', 'ip', 'os', 'username', 'privilege'].includes(
                      key
                    )
                )
                .map(([key, value]) => (
                  <div key={key} className="flex justify-between">
                    <span className="text-dark-text-secondary text-sm">
                      {key}:
                    </span>
                    <span className="text-dark-text-primary text-sm">
                      {String(value)}
                    </span>
                  </div>
                ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
