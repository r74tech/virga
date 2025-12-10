import { create } from 'zustand';
import type { Message, Conversation } from '../types/chat';
import {
  saveMessage,
  loadMessages,
  saveConversation,
  loadConversations,
  deleteConversation,
  clearAllData,
} from '../utils/db';

interface ChatState {
  conversations: Conversation[];
  currentConversationId: string | null;
  messages: Message[];
  isTyping: boolean;

  loadConversations: () => Promise<void>;
  createConversation: (title?: string) => Promise<string>;
  selectConversation: (conversationId: string | null) => Promise<void>;
  deleteConversation: (conversationId: string) => Promise<void>;
  updateConversationTitle: (conversationId: string, title: string) => Promise<void>;

  addMessage: (message: Omit<Message, 'id' | 'timestamp' | 'conversationId'>) => Promise<string>;
  updateMessage: (messageId: string, updates: Partial<Message>) => Promise<void>;
  loadMessages: (conversationId?: string) => Promise<void>;
  setTyping: (isTyping: boolean) => void;
  clearAllData: () => Promise<void>;
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  currentConversationId: null,
  messages: [],
  isTyping: false,

  loadConversations: async () => {
    const conversations = await loadConversations();
    set({ conversations });
  },

  createConversation: async (title?: string) => {
    const id = `conv-${Date.now()}`;
    const now = new Date();
    const conversation: Conversation = {
      id,
      title: title || 'New conversation',
      createdAt: now,
      updatedAt: now,
      messageCount: 0,
    };

    await saveConversation(conversation);
    set((state) => ({
      conversations: [conversation, ...state.conversations],
      currentConversationId: id,
      messages: [],
    }));

    return id;
  },

  selectConversation: async (conversationId: string | null) => {
    if (conversationId === null) {
      set({ currentConversationId: null, messages: [] });
      return;
    }

    set({ currentConversationId: conversationId });
    const messages = await loadMessages(conversationId);
    set({ messages });
  },

  deleteConversation: async (conversationId: string) => {
    await deleteConversation(conversationId);

    set((state) => {
      const newConversations = state.conversations.filter((c) => c.id !== conversationId);
      const wasSelected = state.currentConversationId === conversationId;

      return {
        conversations: newConversations,
        currentConversationId: wasSelected ? null : state.currentConversationId,
        messages: wasSelected ? [] : state.messages,
      };
    });
  },

  updateConversationTitle: async (conversationId: string, title: string) => {
    const { conversations } = get();
    const conversation = conversations.find((c) => c.id === conversationId);

    if (conversation) {
      const updated: Conversation = {
        ...conversation,
        title,
        updatedAt: new Date(),
      };

      await saveConversation(updated);

      set((state) => ({
        conversations: state.conversations.map((c) =>
          c.id === conversationId ? updated : c
        ),
      }));
    }
  },

  addMessage: async (messageData) => {
    const { currentConversationId, conversations } = get();

    let conversationId = currentConversationId;

    if (!conversationId) {
      conversationId = await get().createConversation();
    }

    const message: Message = {
      ...messageData,
      id: `msg-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      conversationId,
      timestamp: new Date(),
    };

    await saveMessage(message);
    set((state) => ({ messages: [...state.messages, message] }));

    const conversation = conversations.find((c) => c.id === conversationId);
    if (conversation) {
      const preview = messageData.content.substring(0, 100);
      const updated: Conversation = {
        ...conversation,
        updatedAt: new Date(),
        messageCount: conversation.messageCount + 1,
        preview,
      };

      await saveConversation(updated);

      set((state) => ({
        conversations: state.conversations
          .map((c) => (c.id === conversationId ? updated : c))
          .sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime()),
      }));
    }

    return message.id;
  },

  updateMessage: async (messageId, updates) => {
    const { messages } = get();
    const messageIndex = messages.findIndex((m) => m.id === messageId);

    if (messageIndex === -1) {
      return;
    }

    const updatedMessage = {
      ...messages[messageIndex],
      ...updates,
    };

    await saveMessage(updatedMessage);

    set((state) => ({
      messages: state.messages.map((m) =>
        m.id === messageId ? updatedMessage : m
      ),
    }));
  },

  loadMessages: async (conversationId?: string) => {
    const messages = await loadMessages(conversationId);
    set({ messages });
  },

  setTyping: (isTyping) => set({ isTyping }),

  clearAllData: async () => {
    await clearAllData();
    set({
      conversations: [],
      currentConversationId: null,
      messages: [],
      isTyping: false,
    });
  },
}));
