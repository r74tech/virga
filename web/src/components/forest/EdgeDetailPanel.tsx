import { X } from 'lucide-react';
import type { Edge } from '../../types/graph';
import { format } from 'date-fns';

interface EdgeDetailPanelProps {
  edge: Edge;
  onClose: () => void;
}

export const EdgeDetailPanel = ({ edge, onClose }: EdgeDetailPanelProps) => {
  const getTypeLabel = (type: string) => {
    switch (type) {
      case 'c2_connection':
        return 'C2 Connection';
      case 'lateral_movement':
        return 'Lateral Movement';
      case 'data_flow':
        return 'Data Flow';
      default:
        return type;
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'c2_connection':
        return 'text-blue-500';
      case 'lateral_movement':
        return 'text-amber-500';
      case 'data_flow':
        return 'text-cyan-500';
      default:
        return 'text-gray-400';
    }
  };

  return (
    <div className="absolute top-4 right-4 w-80 bg-dark-bg-tertiary border border-gray-700 rounded-lg shadow-xl overflow-hidden">
      <div className="flex items-center justify-between p-4 border-b border-gray-700">
        <h3 className="text-lg font-semibold text-dark-text-primary">
          Edge Details
        </h3>
        <button
          onClick={onClose}
          className="p-1 hover:bg-dark-bg-secondary rounded transition-colors"
        >
          <X className="w-5 h-5 text-dark-text-secondary" />
        </button>
      </div>

      <div className="p-4 space-y-4 max-h-[600px] overflow-y-auto">
        {edge.label && (
          <div>
            <label className="text-sm text-dark-text-secondary">Label</label>
            <p className="text-dark-text-primary font-medium">{edge.label}</p>
          </div>
        )}

        <div>
          <label className="text-sm text-dark-text-secondary">Type</label>
          <p className={`font-medium ${getTypeColor(edge.type)}`}>
            {getTypeLabel(edge.type)}
          </p>
        </div>

        <div>
          <label className="text-sm text-dark-text-secondary">Source</label>
          <p className="text-dark-text-primary font-mono text-sm">
            {edge.source}
          </p>
        </div>

        <div>
          <label className="text-sm text-dark-text-secondary">Target</label>
          <p className="text-dark-text-primary font-mono text-sm">
            {edge.target}
          </p>
        </div>

        {edge.weight !== undefined && (
          <div>
            <label className="text-sm text-dark-text-secondary">Weight</label>
            <p className="text-dark-text-primary">{edge.weight}</p>
          </div>
        )}

        {edge.properties && (
          <>
            {edge.properties.bandwidth && (
              <div>
                <label className="text-sm text-dark-text-secondary">
                  Bandwidth
                </label>
                <p className="text-dark-text-primary">
                  {edge.properties.bandwidth}
                </p>
              </div>
            )}

            {edge.properties.latency !== undefined && (
              <div>
                <label className="text-sm text-dark-text-secondary">
                  Latency
                </label>
                <p className="text-dark-text-primary">
                  {edge.properties.latency} ms
                </p>
              </div>
            )}
          </>
        )}

        <div>
          <label className="text-sm text-dark-text-secondary">Created At</label>
          <p className="text-dark-text-primary text-sm">
            {format(edge.createdAt, 'yyyy-MM-dd HH:mm:ss')}
          </p>
        </div>

        {edge.metadata && (
          <>
            {edge.metadata.protocol && (
              <div>
                <label className="text-sm text-dark-text-secondary">
                  Protocol
                </label>
                <p className="text-dark-text-primary">{edge.metadata.protocol}</p>
              </div>
            )}

            {edge.metadata.port && (
              <div>
                <label className="text-sm text-dark-text-secondary">Port</label>
                <p className="text-dark-text-primary font-mono">
                  {edge.metadata.port}
                </p>
              </div>
            )}

            {edge.metadata.established && (
              <div>
                <label className="text-sm text-dark-text-secondary">
                  Established At
                </label>
                <p className="text-dark-text-primary text-sm">
                  {format(
                    new Date(edge.metadata.established),
                    'yyyy-MM-dd HH:mm:ss'
                  )}
                </p>
              </div>
            )}
          </>
        )}

        {edge.properties && Object.keys(edge.properties).length > 0 && (
          <div>
            <label className="text-sm text-dark-text-secondary mb-2 block">
              Additional Properties
            </label>
            <div className="bg-dark-bg-secondary p-3 rounded space-y-2">
              {Object.entries(edge.properties)
                .filter(([key]) => !['bandwidth', 'latency'].includes(key))
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
