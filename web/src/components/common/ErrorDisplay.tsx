import { AlertCircle } from 'lucide-react';

interface ErrorDisplayProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  fullScreen?: boolean;
  variant?: 'inline' | 'card';
}

export const ErrorDisplay = ({
  title = 'Error',
  message,
  onRetry,
  fullScreen = false,
  variant = 'card',
}: ErrorDisplayProps) => {
  const Container = fullScreen ? 'div' : 'div';
  const containerClass = fullScreen
    ? 'h-full flex items-center justify-center'
    : '';

  if (variant === 'inline') {
    return (
      <div className="p-4 bg-red-500/10 border border-red-500/50 rounded-lg">
        <div className="flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-red-400 text-sm font-medium mb-1">{title}</p>
            <p className="text-red-300 text-xs">{message}</p>
            {onRetry && (
              <button
                type="button"
                onClick={onRetry}
                className="mt-2 text-xs text-red-400 hover:text-red-300 underline"
              >
                Retry
              </button>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <Container className={containerClass}>
      <div className="max-w-md p-6 bg-red-500/10 border border-red-500/50 rounded-lg">
        <div className="flex items-start gap-3">
          <AlertCircle className="w-6 h-6 text-red-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-red-400 text-lg font-medium mb-2">{title}</p>
            <p className="text-red-300 text-sm mb-4">{message}</p>
            {onRetry && (
              <button
                type="button"
                onClick={onRetry}
                className="px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors text-sm"
              >
                Retry
              </button>
            )}
          </div>
        </div>
      </div>
    </Container>
  );
};
