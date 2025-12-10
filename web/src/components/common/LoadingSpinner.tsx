import { Loader2 } from 'lucide-react';

interface LoadingSpinnerProps {
  message?: string;
  size?: 'sm' | 'md' | 'lg';
  fullScreen?: boolean;
}

export const LoadingSpinner = ({
  message = 'Loading...',
  size = 'md',
  fullScreen = false,
}: LoadingSpinnerProps) => {
  const sizeClasses = {
    sm: 'w-4 h-4',
    md: 'w-8 h-8',
    lg: 'w-12 h-12',
  };

  const Container = fullScreen ? 'div' : 'div';
  const containerClass = fullScreen
    ? 'h-full flex items-center justify-center'
    : 'flex items-center justify-center';

  return (
    <Container className={containerClass}>
      <div className="text-center">
        <Loader2
          className={`${sizeClasses[size]} text-primary-500 animate-spin mx-auto mb-2`}
        />
        <p className="text-dark-text-secondary text-sm">{message}</p>
      </div>
    </Container>
  );
};
