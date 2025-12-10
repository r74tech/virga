import { useState, useRef, useEffect, useMemo, type KeyboardEvent } from 'react';
import { Terminal as TerminalIcon, Plus, X } from 'lucide-react';
import type { Session } from '../../types/session';
import { useSettingsStore } from '../../stores/settingsStore';
import { useTerminalStore, type Terminal } from '../../stores/terminalStore';

interface SessionTerminalProps {
  session: Session;
}

export const SessionTerminal = ({ session }: SessionTerminalProps) => {
  const terminalSettings = useSettingsStore((state) => state.settings.terminal);

  // Use terminal store for session-specific state
  const {
    terminals,
    sessionTerminalIds: allSessionTerminalIds,
    activeTerminalIds,
    createTerminal,
    setActiveTerminal,
    closeTerminal,
    executeCommand: executeCommandFromStore,
  } = useTerminalStore();

  // Get session-specific data (memoized to prevent infinite loops)
  const sessionTerminalIds = useMemo(
    () => allSessionTerminalIds[session.id] || [],
    [allSessionTerminalIds, session.id]
  );
  const activeTerminalId = activeTerminalIds[session.id];

  // Derive session terminals and active terminal from state
  const sessionTerminals = useMemo(
    () => sessionTerminalIds.map((id) => terminals[id]).filter((t): t is Terminal => t !== undefined),
    [sessionTerminalIds, terminals]
  );
  const activeTerminal = useMemo(
    () => (activeTerminalId ? terminals[activeTerminalId] : null),
    [activeTerminalId, terminals]
  );

  const [currentInput, setCurrentInput] = useState('');
  const terminalRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Initialize first terminal with welcome message
  useEffect(() => {
    if (sessionTerminalIds.length === 0) {
      createTerminal(session.id, [
        `Connected to ${session.hostname} (${session.ip})`,
        `User: ${session.username} | Privilege: ${session.privilege || 'User'}`,
        session.connectionType === 'direct'
          ? 'Connection: Direct C2'
          : `Connection: Pivot via ${session.pivotChain?.join(' -> ')}`,
        '',
      ]);
    }
  }, [session.id, sessionTerminalIds.length, session.hostname, session.ip, session.username, session.privilege, session.connectionType, session.pivotChain, createTerminal]);

  // Auto-scroll to bottom when entries change
  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
    }
  }, [activeTerminal?.entries.length, currentInput]);

  // Focus input when terminal is clicked (but not when selecting text)
  useEffect(() => {
    const handleTerminalClick = () => {
      // Don't focus if user is selecting text
      const selection = window.getSelection();
      if (selection && selection.toString().length > 0) {
        return;
      }

      // Focus input on click
      inputRef.current?.focus();
    };

    const terminalElement = terminalRef.current;
    if (terminalElement) {
      terminalElement.addEventListener('click', handleTerminalClick);
      return () => terminalElement.removeEventListener('click', handleTerminalClick);
    }
  }, []);

  const executeCommand = async (cmd: string) => {
    if (!cmd.trim() || !activeTerminal || activeTerminal.isExecuting) return;

    setCurrentInput('');
    await executeCommandFromStore(activeTerminal.id, cmd);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !activeTerminal?.isExecuting) {
      e.preventDefault();
      executeCommand(currentInput);
    }
  };

  const handleNewTerminal = () => {
    createTerminal(session.id, [
      `Connected to ${session.hostname} (${session.ip})`,
      '',
    ]);
  };

  const handleCloseTerminal = (terminalId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (sessionTerminals.length > 1) {
      closeTerminal(terminalId);
    }
  };

  if (!activeTerminal) {
    return (
      <div className="flex items-center justify-center h-full bg-dark-bg">
        <div className="text-dark-text-secondary">Loading terminal...</div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full bg-dark-bg">
      {/* Terminal Tabs */}
      <div className="flex items-center gap-1 px-2 py-1 border-b border-gray-700 bg-dark-bg-secondary overflow-x-auto">
        {sessionTerminals.map((terminal) => (
          <div
            key={terminal.id}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-t text-sm transition-colors ${terminal.id === activeTerminal.id
              ? 'bg-dark-bg text-dark-text-primary border-t border-l border-r border-gray-700'
              : 'bg-dark-bg-tertiary text-dark-text-secondary hover:bg-dark-bg cursor-pointer'
              }`}
          >
            <button
              type="button"
              onClick={() => setActiveTerminal(session.id, terminal.id)}
              className="flex items-center gap-2 flex-1"
            >
              <TerminalIcon className="w-3 h-3" />
              <span className="max-w-32 truncate">{terminal.name}</span>
            </button>
            {sessionTerminals.length > 1 && (
              <button
                type="button"
                onClick={(e) => handleCloseTerminal(terminal.id, e)}
                className="ml-1 hover:text-red-400 transition-colors"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </div>
        ))}
        <button
          type="button"
          onClick={handleNewTerminal}
          className="flex items-center gap-1 px-2 py-1.5 text-xs text-dark-text-secondary hover:text-dark-text-primary hover:bg-dark-bg rounded transition-colors"
          title="New Terminal"
        >
          <Plus className="w-4 h-4" />
        </button>
      </div>

      {/* Terminal Content */}
      <div
        ref={terminalRef}
        className="flex-1 overflow-y-auto p-3 font-mono cursor-text"
        style={{
          fontFamily: terminalSettings.fontFamily,
          fontSize: `${terminalSettings.fontSize}px`,
        }}
      >
        {activeTerminal.entries.map((entry, index) => {
          // Determine if this entry was in shell mode by checking previous system messages
          const isInShellMode = activeTerminal.entries
            .slice(0, index)
            .reverse()
            .find(e => e.type === 'system' &&
              (e.content.includes('Starting interactive shell') ||
                e.content.includes('Exiting interactive shell')))
            ?.content.includes('Starting interactive shell') ?? false;

          return (
            <div key={entry.id} className="mb-1 font-mono" style={{ fontFamily: terminalSettings.fontFamily, fontSize: `${terminalSettings.fontSize}px` }}>
              {entry.type === 'input' ? (
                <div className="flex gap-2">
                  <span className="text-primary-500 select-none">
                    {isInShellMode
                      ? 'shell>'
                      : `virga (${session.id.slice(0, 8)})>`}
                  </span>
                  <span className="text-dark-text-primary select-text">{entry.content}</span>
                </div>
              ) : entry.type === 'error' ? (
                <div className="text-red-400 whitespace-pre-wrap select-text">
                  {entry.content}
                </div>
              ) : entry.type === 'system' ? (
                <div className="text-gray-400 whitespace-pre-wrap select-text">
                  {entry.content}
                </div>
              ) : (
                <div className="text-gray-300 whitespace-pre-wrap select-text">
                  {entry.content}
                </div>
              )}
            </div>
          );
        })}

        {/* Current input line - like a real terminal */}
        {!activeTerminal.isExecuting && (
          <div className="flex gap-2">
            <span className="text-primary-500 select-none">
              {activeTerminal.isShellMode
                ? 'shell>'
                : `virga (${session.id.slice(0, 8)})>`}
            </span>
            <div className="flex-1 relative">
              <input
                ref={inputRef}
                type="text"
                value={currentInput}
                onChange={(e) => setCurrentInput(e.target.value)}
                onKeyDown={handleKeyDown}
                className="w-full bg-transparent border-none outline-none text-dark-text-primary font-mono p-0 m-0"
                style={{
                  fontFamily: terminalSettings.fontFamily,
                  fontSize: `${terminalSettings.fontSize}px`,
                }}
                autoFocus
              />
            </div>
          </div>
        )}

        {activeTerminal.isExecuting && (
          <div className="flex gap-2 text-dark-text-secondary">
            <span className="animate-pulse">Executing...</span>
          </div>
        )}
      </div>
    </div>
  );
};
