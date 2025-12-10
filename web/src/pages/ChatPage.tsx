import { useEffect, useRef, useState } from 'react';
import { Menu } from 'lucide-react';
import { Message } from '../components/chat/Message';
import { MessageInput } from '../components/chat/MessageInput';
import { ConversationSidebar } from '../components/chat/ConversationSidebar';
import { SessionSelector } from '../components/chat/SessionSelector';
import { useChatStore } from '../stores/chatStore';
import { useSettingsStore } from '../stores/settingsStore';
import { apiClient } from '../services/apiClient';
import { SessionsApi } from '../services/api/sessions';
import type { Session } from '../types/session';

export const ChatPage = () => {
  const conversations = useChatStore((state) => state.conversations);
  const currentConversationId = useChatStore(
    (state) => state.currentConversationId
  );
  const messages = useChatStore((state) => state.messages);
  const isTyping = useChatStore((state) => state.isTyping);
  const addMessage = useChatStore((state) => state.addMessage);
  const updateMessage = useChatStore((state) => state.updateMessage);
  const setTyping = useChatStore((state) => state.setTyping);
  const loadConversations = useChatStore((state) => state.loadConversations);
  const createConversation = useChatStore((state) => state.createConversation);

  const chatTimeout = useSettingsStore((state) => state.settings.session.chatTimeout);

  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [selectedSession, setSelectedSession] = useState<Session | null>(null);
  const [error, setError] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const sessionsApi = new SessionsApi(apiClient);

  useEffect(() => {
    const initConversations = async () => {
      await loadConversations();

      const state = useChatStore.getState();
      if (state.conversations.length === 0 && !state.currentConversationId) {
        await createConversation('First Conversation');
      }
    };

    initConversations();
  }, [loadConversations, createConversation]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  const handleSendMessage = async (content: string) => {
    // Check if session is not selected
    if (!selectedSession) {
      setError('Please select a session first');
      return;
    }

    setError(null);

    // Add user message
    await addMessage({
      type: 'user',
      content,
    });

    setTyping(true);

    // Create assistant message (for progress tracking)
    const messageId = await addMessage({
      type: 'assistant',
      content: 'Processing...',
      metadata: {},
    });

    try {
      // Send chat message (async, only taskId is returned)
      const { taskId } = await sessionsApi.sendChatMessage(
        selectedSession.id,
        content
      );

      // Poll for results
      let partialOutputs: string[] = [];
      const pollInterval = 500; // 500ms
      const maxAttempts = (chatTimeout * 60 * 1000) / pollInterval; // chatTimeout minutes
      let attempts = 0;

      const pollResults = setInterval(async () => {
        attempts++;

        try {
          // Get task result
          const task = await sessionsApi.getTaskResult(
            selectedSession.id,
            taskId
          );

          if (task?.results) {
            // Extract only partial: true results (intermediate results)
            // Exclude partial: false results (initial message and final result)
            const partialResults = task.results.filter((r) => r.partial);
            const newPartialOutputs = partialResults.map((r) => r.output);

            // Display if results are updated
            if (newPartialOutputs.length > partialOutputs.length) {
              partialOutputs = newPartialOutputs;

              await updateMessage(messageId, {
                content: '_Processing..._',
                metadata: {
                  taskId,
                  status: task.status,
                  partialResults: partialOutputs,
                  isProcessing: true,
                },
              });
            }
          }

          // Exit if task is complete
          if (task.isComplete) {
            clearInterval(pollResults);

            // Get final result (last entry with partial: false)
            const finalResults = task.results.filter((r) => !r.partial);
            const finalResult = finalResults.length > 0
              ? finalResults[finalResults.length - 1]
              : (task.results.length > 0 ? task.results[task.results.length - 1] : null);
            const finalOutput = finalResult
              ? finalResult.output
              : partialOutputs.join('\n\n');

            await updateMessage(messageId, {
              content: finalOutput || 'No output',
              metadata: { taskId, status: task.status },
            });

            if (finalResult?.error) {
              setError(finalResult.error);
            }

            setTyping(false);
          }

          // Timeout check
          if (attempts >= maxAttempts) {
            clearInterval(pollResults);
            await updateMessage(messageId, {
              content: `${partialOutputs.join('\n\n')}\n\n_Task timed out_`,
              metadata: { taskId, status: 'timeout' },
            });
            setTyping(false);
          }
        } catch (pollErr) {
          console.error('Polling error:', pollErr);
        }
      }, pollInterval);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'An error occurred';
      setError(errorMessage);

      // Update error message
      await updateMessage(messageId, {
        type: 'system',
        content: `Error: ${errorMessage}`,
        metadata: {},
      });
      setTyping(false);
    }
  };

  const currentConversation = conversations.find(
    (c) => c.id === currentConversationId
  );

  return (
    <div className="h-full flex">
      <ConversationSidebar
        isOpen={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
      />

      <div className="flex-1 flex flex-col">
        <div className="p-4 border-b border-gray-700 bg-dark-bg-secondary">
          <div className="flex items-center gap-3 mb-3">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="md:hidden p-2 hover:bg-dark-bg-tertiary rounded-lg transition-colors"
              type="button"
            >
              <Menu className="w-5 h-5 text-dark-text-secondary" />
            </button>
            <div className="flex-1">
              <h1 className="text-xl font-bold text-dark-text-primary">
                {currentConversation?.title || 'AI Chat Interface'}
              </h1>
              <p className="text-sm text-dark-text-secondary mt-1">
                Control Virga C2 using natural language
              </p>
            </div>
          </div>

          <SessionSelector
            selectedSessionId={selectedSession?.id || null}
            onSessionSelect={setSelectedSession}
          />

          {error && (
            <div className="mt-3 p-2 bg-red-900/20 border border-red-700 rounded-lg text-sm text-red-400">
              {error}
            </div>
          )}
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {messages.length === 0 ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <p className="text-dark-text-secondary">
                  Send a message to start the conversation
                </p>
                {!selectedSession && (
                  <p className="text-sm text-yellow-500 mt-2">
                    Please select a session first
                  </p>
                )}
              </div>
            </div>
          ) : (
            <>
              {messages.map((message) => (
                <Message key={message.id} message={message} />
              ))}

              {isTyping && (
                <div className="p-4 rounded-lg border bg-dark-bg-secondary border-gray-700">
                  <div className="flex items-center gap-3">
                    <div className="flex gap-1">
                      <div className="w-2 h-2 bg-primary-500 rounded-full animate-bounce" />
                      <div
                        className="w-2 h-2 bg-primary-500 rounded-full animate-bounce"
                        style={{ animationDelay: '0.1s' }}
                      />
                      <div
                        className="w-2 h-2 bg-primary-500 rounded-full animate-bounce"
                        style={{ animationDelay: '0.2s' }}
                      />
                    </div>
                    <span className="text-sm text-dark-text-secondary">
                      AI is processing...
                    </span>
                  </div>
                </div>
              )}

              <div ref={messagesEndRef} />
            </>
          )}
        </div>

        <MessageInput
          onSend={handleSendMessage}
          disabled={isTyping || !selectedSession}
          placeholder={
            selectedSession
              ? `Example: Get process list on ${selectedSession.hostname}`
              : 'Please select a session first'
          }
        />
      </div>
    </div>
  );
};
