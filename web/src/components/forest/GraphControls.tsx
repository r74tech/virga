import { RotateCcw, ZoomIn, ZoomOut, Maximize2 } from 'lucide-react';

interface GraphControlsProps {
  onResetLayout: () => void;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onFit: () => void;
}

export const GraphControls = ({
  onResetLayout,
  onZoomIn,
  onZoomOut,
  onFit,
}: GraphControlsProps) => {
  return (
    <div className="absolute bottom-4 left-4 flex flex-col gap-2 bg-dark-bg-tertiary border border-gray-700 rounded-lg p-2">
      <button
        onClick={onFit}
        className="p-2 hover:bg-dark-bg-secondary rounded transition-colors group"
        title="Fit to screen"
      >
        <Maximize2 className="w-5 h-5 text-dark-text-secondary group-hover:text-dark-text-primary" />
      </button>

      <button
        onClick={onZoomIn}
        className="p-2 hover:bg-dark-bg-secondary rounded transition-colors group"
        title="Zoom in"
      >
        <ZoomIn className="w-5 h-5 text-dark-text-secondary group-hover:text-dark-text-primary" />
      </button>

      <button
        onClick={onZoomOut}
        className="p-2 hover:bg-dark-bg-secondary rounded transition-colors group"
        title="Zoom out"
      >
        <ZoomOut className="w-5 h-5 text-dark-text-secondary group-hover:text-dark-text-primary" />
      </button>

      <div className="border-t border-gray-700 my-1" />

      <button
        onClick={onResetLayout}
        className="p-2 hover:bg-dark-bg-secondary rounded transition-colors group"
        title="Reset layout"
      >
        <RotateCcw className="w-5 h-5 text-dark-text-secondary group-hover:text-dark-text-primary" />
      </button>
    </div>
  );
};
