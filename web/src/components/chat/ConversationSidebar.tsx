import { Plus, MessageSquare, Trash2, Edit2, Check, X } from 'lucide-react';
import { useState } from 'react';
import { useChatStore } from '../../stores/chatStore';
import { format } from 'date-fns';
import type { Conversation } from '../../types/chat';

interface ConversationSidebarProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ConversationSidebar = ({
  isOpen,
  onClose,
}: ConversationSidebarProps) => {
  const conversations = useChatStore((state) => state.conversations);
  const currentConversationId = useChatStore(
    (state) => state.currentConversationId
  );
  const createConversation = useChatStore((state) => state.createConversation);
  const selectConversation = useChatStore((state) => state.selectConversation);
  const deleteConversation = useChatStore((state) => state.deleteConversation);
  const updateConversationTitle = useChatStore(
    (state) => state.updateConversationTitle
  );

  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');

  const handleNewConversation = async () => {
    await createConversation();
  };

  const handleSelectConversation = async (conversationId: string) => {
    await selectConversation(conversationId);
    if (window.innerWidth < 768) {
      onClose();
    }
  };

  const handleDeleteConversation = async (
    e: React.MouseEvent,
    conversationId: string
  ) => {
    e.stopPropagation();
    if (confirm('Are you sure you want to delete this conversation?')) {
      await deleteConversation(conversationId);
    }
  };

  const handleStartEdit = (
    e: React.MouseEvent,
    conversation: Conversation
  ) => {
    e.stopPropagation();
    setEditingId(conversation.id);
    setEditTitle(conversation.title);
  };

  const handleSaveEdit = async (conversationId: string) => {
    if (editTitle.trim()) {
      await updateConversationTitle(conversationId, editTitle.trim());
    }
    setEditingId(null);
    setEditTitle('');
  };

  const handleCancelEdit = () => {
    setEditingId(null);
    setEditTitle('');
  };

  return (
    <>
      {isOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 z-40 md:hidden"
          onClick={onClose}
        />
      )}

      <div
        className={`fixed md:relative top-0 left-0 h-full w-80 bg-dark-bg-secondary border-r border-gray-700 flex flex-col z-50 transition-transform duration-300 ${
          isOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'
        }`}
      >
        <div className="p-4 border-b border-gray-700">
          <button
            onClick={handleNewConversation}
            className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 rounded-lg transition-colors"
          >
            <Plus className="w-5 h-5" />
            <span className="font-medium">New Conversation</span>
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          {conversations.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-dark-text-secondary">
              <MessageSquare className="w-12 h-12 mb-2 opacity-50" />
              <p className="text-sm">No conversations</p>
              <p className="text-xs mt-1">Start a new conversation</p>
            </div>
          ) : (
            conversations.map((conversation) => (
              <div
                key={conversation.id}
                onClick={() => handleSelectConversation(conversation.id)}
                className={`group relative p-3 rounded-lg cursor-pointer transition-colors ${
                  currentConversationId === conversation.id
                    ? 'bg-dark-bg-tertiary border border-primary-600'
                    : 'hover:bg-dark-bg-tertiary border border-transparent'
                }`}
              >
                {editingId === conversation.id ? (
                  <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
                    <input
                      type="text"
                      value={editTitle}
                      onChange={(e) => setEditTitle(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          handleSaveEdit(conversation.id);
                        } else if (e.key === 'Escape') {
                          handleCancelEdit();
                        }
                      }}
                      className="flex-1 px-2 py-1 bg-dark-bg border border-gray-600 rounded text-sm text-dark-text-primary focus:outline-none focus:border-primary-500"
                      autoFocus
                    />
                    <button
                      onClick={() => handleSaveEdit(conversation.id)}
                      className="p-1 hover:bg-dark-bg-secondary rounded"
                    >
                      <Check className="w-4 h-4 text-green-500" />
                    </button>
                    <button
                      onClick={handleCancelEdit}
                      className="p-1 hover:bg-dark-bg-secondary rounded"
                    >
                      <X className="w-4 h-4 text-red-500" />
                    </button>
                  </div>
                ) : (
                  <>
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex-1 min-w-0">
                        <h3 className="font-medium text-dark-text-primary text-sm truncate">
                          {conversation.title}
                        </h3>
                        {conversation.preview && (
                          <p className="text-xs text-dark-text-secondary mt-1 line-clamp-2">
                            {conversation.preview}
                          </p>
                        )}
                        <div className="flex items-center gap-3 mt-2 text-xs text-dark-text-secondary">
                          <span>{conversation.messageCount} messages</span>
                          <span>
                            {format(conversation.updatedAt, 'M/d HH:mm')}
                          </span>
                        </div>
                      </div>

                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          onClick={(e) => handleStartEdit(e, conversation)}
                          className="p-1.5 hover:bg-dark-bg rounded"
                          title="Edit Title"
                        >
                          <Edit2 className="w-3.5 h-3.5 text-dark-text-secondary" />
                        </button>
                        <button
                          onClick={(e) =>
                            handleDeleteConversation(e, conversation.id)
                          }
                          className="p-1.5 hover:bg-dark-bg rounded"
                          title="Delete"
                        >
                          <Trash2 className="w-3.5 h-3.5 text-red-400" />
                        </button>
                      </div>
                    </div>
                  </>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </>
  );
};
