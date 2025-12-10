import { useState, KeyboardEvent } from 'react';
import { Send } from 'lucide-react';

interface MessageInputProps {
  onSend: (message: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

export const MessageInput = ({
  onSend,
  disabled = false,
  placeholder = 'Type a message...',
}: MessageInputProps) => {
  const [input, setInput] = useState('');

  const handleSend = () => {
    if (input.trim() && !disabled) {
      onSend(input.trim());
      setInput('');
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="border-t border-gray-700 bg-dark-bg-secondary p-4">
      <div className="flex gap-3">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled}
          rows={1}
          className="flex-1 px-4 py-2 bg-dark-bg-tertiary border border-gray-700 rounded-lg text-dark-text-primary placeholder-dark-text-secondary focus:outline-none focus:border-primary-500 resize-none disabled:opacity-50 disabled:cursor-not-allowed"
          style={{
            minHeight: '44px',
            maxHeight: '200px',
          }}
        />
        <button
          onClick={handleSend}
          disabled={disabled || !input.trim()}
          className="px-4 py-2 bg-primary-500 hover:bg-primary-600 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg transition-colors flex items-center gap-2"
        >
          <Send className="w-5 h-5" />
          <span>Send</span>
        </button>
      </div>
      <div className="mt-2 text-xs text-dark-text-secondary">
        Press Enter to send, Shift+Enter for new line
      </div>
    </div>
  );
};
