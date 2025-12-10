import { useState } from 'react';
import { User, Bot, Terminal, AlertCircle, ChevronDown, ChevronRight } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { Message as MessageType } from '../../types/chat';
import { format } from 'date-fns';

interface MessageProps {
  message: MessageType;
}

export const Message = ({ message }: MessageProps) => {
  const [isExpanded, setIsExpanded] = useState(false);

  // Partial results processing
  const partialResults = (message.metadata?.partialResults as string[]) || [];
  const isProcessing = message.metadata?.isProcessing === true;

  // Iteration results and other messages classification
  const iterations = partialResults.filter((o) => o.startsWith('=== Iteration'));
  const others = partialResults.filter((o) => !o.startsWith('=== Iteration'));
  const getIcon = () => {
    switch (message.type) {
      case 'user':
        return <User className="w-5 h-5" />;
      case 'assistant':
        return <Bot className="w-5 h-5" />;
      case 'system':
        return <AlertCircle className="w-5 h-5" />;
      case 'command':
        return <Terminal className="w-5 h-5" />;
      default:
        return <Bot className="w-5 h-5" />;
    }
  };

  const getBackgroundColor = () => {
    switch (message.type) {
      case 'user':
        return 'bg-primary-500/10 border-primary-500/30';
      case 'assistant':
        return 'bg-dark-bg-secondary border-gray-700';
      case 'system':
        return 'bg-amber-500/10 border-amber-500/30';
      case 'command':
        return 'bg-green-500/10 border-green-500/30';
      default:
        return 'bg-dark-bg-secondary border-gray-700';
    }
  };

  const getIconColor = () => {
    switch (message.type) {
      case 'user':
        return 'text-primary-500';
      case 'assistant':
        return 'text-blue-400';
      case 'system':
        return 'text-amber-400';
      case 'command':
        return 'text-green-400';
      default:
        return 'text-gray-400';
    }
  };

  return (
    <div className={`p-4 rounded-lg border ${getBackgroundColor()}`}>
      <div className="flex items-start gap-3">
        <div className={`mt-1 ${getIconColor()}`}>{getIcon()}</div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2">
            <span className="font-semibold text-dark-text-primary capitalize">
              {message.type}
            </span>
            <span className="text-xs text-dark-text-secondary">
              {format(message.timestamp, 'HH:mm:ss')}
            </span>
          </div>

          {/* Partial results display */}
          {isProcessing && partialResults.length > 0 && (
            <div className="mb-4">
              {/* Other messages */}
              {others.length > 0 && (
                <div className="mb-3 text-sm text-dark-text-secondary">
                  {others.map((msg, idx) => (
                    <div key={idx} className="mb-1">
                      {msg}
                    </div>
                  ))}
                </div>
              )}

              {/* Iteration results */}
              {iterations.length > 0 && (
                <div>
                  {/* Previous iteration (collapsible) */}
                  {iterations.length > 1 && (
                    <div className="mb-2">
                      <button
                        onClick={() => setIsExpanded(!isExpanded)}
                        className="flex items-center gap-2 text-sm text-dark-text-secondary hover:text-dark-text-primary transition-colors"
                        type="button"
                      >
                        {isExpanded ? (
                          <ChevronDown className="w-4 h-4" />
                        ) : (
                          <ChevronRight className="w-4 h-4" />
                        )}
                        <span>Previous iteration ({iterations.length - 1} times)</span>
                      </button>
                      {isExpanded && (
                        <div className="mt-2 pl-6 space-y-3 border-l-2 border-gray-700">
                          {iterations.slice(0, -1).map((iter, idx) => (
                            <div key={idx} className="text-sm prose prose-invert prose-sm max-w-none">
                              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                                {iter}
                              </ReactMarkdown>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}

                  {/* Latest iteration */}
                  <div className="prose prose-invert prose-sm max-w-none">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>
                      {iterations[iterations.length - 1]}
                    </ReactMarkdown>
                  </div>
                </div>
              )}
            </div>
          )}

          <div className="prose prose-invert prose-sm max-w-none">
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                code: (props) => {
                  const { children } = props;
                  const isInline = !String(children).includes('\n');
                  return isInline ? (
                    <code
                      className="bg-dark-bg-tertiary px-1.5 py-0.5 rounded text-sm font-mono text-primary-400"
                      {...props}
                    >
                      {children}
                    </code>
                  ) : (
                    <code
                      className="block bg-dark-bg-tertiary p-3 rounded text-sm font-mono overflow-x-auto"
                      {...props}
                    >
                      {children}
                    </code>
                  );
                },
                pre: ({ children }) => <div className="my-2">{children}</div>,
                p: ({ children }) => (
                  <p className="text-dark-text-primary mb-2 last:mb-0">
                    {children}
                  </p>
                ),
                ul: ({ children }) => (
                  <ul className="list-disc list-inside text-dark-text-primary space-y-1 my-2">
                    {children}
                  </ul>
                ),
                ol: ({ children }) => (
                  <ol className="list-decimal list-inside text-dark-text-primary space-y-1 my-2">
                    {children}
                  </ol>
                ),
                li: ({ children }) => (
                  <li className="text-dark-text-primary">{children}</li>
                ),
                strong: ({ children }) => (
                  <strong className="font-semibold text-dark-text-primary">
                    {children}
                  </strong>
                ),
                a: ({ href, children }) => (
                  <a
                    href={href}
                    className="text-primary-400 hover:text-primary-300 underline"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {children}
                  </a>
                ),
              }}
            >
              {message.content}
            </ReactMarkdown>
          </div>

          {message.metadata?.progress && (
            <div className="mt-3 pt-3 border-t border-gray-700">
              <div className="text-xs text-dark-text-secondary mb-2">
                Progress:
              </div>
              <div className="bg-dark-bg-tertiary p-3 rounded">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm text-dark-text-primary">
                    Iteration {message.metadata.progress.current_iteration} /{' '}
                    {message.metadata.progress.max_iterations}
                  </span>
                  <span className="text-xs text-dark-text-secondary">
                    {Math.round(
                      (message.metadata.progress.current_iteration /
                        message.metadata.progress.max_iterations) *
                        100
                    )}
                    %
                  </span>
                </div>
                <div className="w-full bg-dark-bg h-2 rounded-full overflow-hidden">
                  <div
                    className="bg-primary-500 h-full transition-all duration-300"
                    style={{
                      width: `${
                        (message.metadata.progress.current_iteration /
                          message.metadata.progress.max_iterations) *
                        100
                      }%`,
                    }}
                  />
                </div>
              </div>
            </div>
          )}

          {message.metadata?.command && (
            <div className="mt-3 pt-3 border-t border-gray-700">
              <div className="text-xs text-dark-text-secondary mb-1">Command:</div>
              <div className="bg-dark-bg-tertiary p-2 rounded font-mono text-sm text-green-400">
                {message.metadata.command}
              </div>
              {message.metadata.result && (
                <>
                  <div className="text-xs text-dark-text-secondary mt-2 mb-1">
                    Result:
                  </div>
                  <div className="bg-dark-bg-tertiary p-2 rounded font-mono text-sm text-dark-text-primary whitespace-pre-wrap max-h-64 overflow-y-auto">
                    {message.metadata.result.output}
                  </div>
                  {message.metadata.result.exitCode !== undefined && (
                    <div className="text-xs text-dark-text-secondary mt-1">
                      Exit code: {message.metadata.result.exitCode}
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
