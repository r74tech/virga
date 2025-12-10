import { useRef, useEffect } from 'react';
import { AlertCircle, Loader2 } from 'lucide-react';
import { ForestGraph, type ForestGraphHandle } from '../components/forest/ForestGraph';
import { NodeDetailPanel } from '../components/forest/NodeDetailPanel';
import { EdgeDetailPanel } from '../components/forest/EdgeDetailPanel';
import { GraphControls } from '../components/forest/GraphControls';
import { useForestStore } from '../stores/forestStore';
import { useSettingsStore } from '../stores/settingsStore';
import { useAutoRefresh } from '../hooks/useAutoRefresh';

export const ForestPage = () => {
  const graphRef = useRef<ForestGraphHandle>(null);
  const selectedNode = useForestStore((state) => state.selectedNode);
  const selectedEdge = useForestStore((state) => state.selectedEdge);
  const setSelectedNode = useForestStore((state) => state.setSelectedNode);
  const setSelectedEdge = useForestStore((state) => state.setSelectedEdge);
  const isLoading = useForestStore((state) => state.isLoading);
  const error = useForestStore((state) => state.error);
  const fetchGraphData = useForestStore((state) => state.fetchGraphData);
  const clearError = useForestStore((state) => state.clearError);

  const isLoaded = useSettingsStore((state) => state.isLoaded);
  const loadSettings = useSettingsStore((state) => state.loadSettings);

  useEffect(() => {
    if (!isLoaded) {
      loadSettings();
    }
  }, [isLoaded, loadSettings]);

  // Auto-refresh graph data every 10 seconds
  useAutoRefresh(fetchGraphData, {
    enabled: true,
    interval: 10000,
  });

  const handleResetLayout = () => {
    graphRef.current?.resetLayout();
  };

  const handleZoomIn = () => {
    graphRef.current?.zoomIn();
  };

  const handleZoomOut = () => {
    graphRef.current?.zoomOut();
  };

  const handleFit = () => {
    graphRef.current?.fit();
  };

  const handleCloseDetails = () => {
    setSelectedNode(null);
    setSelectedEdge(null);
  };

  if (!isLoaded) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="w-8 h-8 text-primary-500 animate-spin mx-auto mb-2" />
          <p className="text-dark-text-secondary">Loading settings...</p>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="w-8 h-8 text-primary-500 animate-spin mx-auto mb-2" />
          <p className="text-dark-text-secondary">Loading graph data...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="max-w-md p-6 bg-red-500/10 border border-red-500/50 rounded-lg">
          <div className="flex items-start gap-3">
            <AlertCircle className="w-6 h-6 text-red-400 flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-red-400 text-lg font-medium mb-2">
                Failed to load graph data
              </p>
              <p className="text-red-300 text-sm mb-4">{error}</p>
              <button
                type="button"
                onClick={() => {
                  clearError();
                  fetchGraphData();
                }}
                className="px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors text-sm"
              >
                Retry
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full relative">
      <ForestGraph ref={graphRef} className="w-full h-full" />

      <GraphControls
        onResetLayout={handleResetLayout}
        onZoomIn={handleZoomIn}
        onZoomOut={handleZoomOut}
        onFit={handleFit}
      />

      {selectedNode && (
        <NodeDetailPanel node={selectedNode} onClose={handleCloseDetails} />
      )}

      {selectedEdge && (
        <EdgeDetailPanel edge={selectedEdge} onClose={handleCloseDetails} />
      )}
    </div>
  );
};
