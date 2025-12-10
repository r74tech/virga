import type { Message, Conversation } from '../types/chat';
import type { Settings } from '../types/settings';

const DB_NAME = 'virga-c2-db';
const DB_VERSION = 4;
const MESSAGES_STORE = 'chat-messages';
const CONVERSATIONS_STORE = 'conversations';
const SETTINGS_STORE = 'settings';
const ZUSTAND_STORE = 'zustand-persist';

let dbPromise: Promise<IDBDatabase> | null = null;

function openDB(): Promise<IDBDatabase> {
  if (dbPromise) return dbPromise;

  dbPromise = new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);

    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;
      const oldVersion = event.oldVersion;

      if (oldVersion < 1) {
        if (!db.objectStoreNames.contains(MESSAGES_STORE)) {
          const store = db.createObjectStore(MESSAGES_STORE, {
            keyPath: 'id',
          });
          store.createIndex('timestamp', 'timestamp', { unique: false });
          store.createIndex('conversationId', 'conversationId', { unique: false });
        }
      }

      if (oldVersion < 2) {
        if (!db.objectStoreNames.contains(CONVERSATIONS_STORE)) {
          const store = db.createObjectStore(CONVERSATIONS_STORE, {
            keyPath: 'id',
          });
          store.createIndex('updatedAt', 'updatedAt', { unique: false });
        }

        if (db.objectStoreNames.contains(MESSAGES_STORE)) {
          const transaction = (event.target as IDBOpenDBRequest).transaction;
          if (transaction) {
            const messageStore = transaction.objectStore(MESSAGES_STORE);
            if (!messageStore.indexNames.contains('conversationId')) {
              messageStore.createIndex('conversationId', 'conversationId', { unique: false });
            }
          }
        }
      }

      if (oldVersion < 3) {
        if (!db.objectStoreNames.contains(SETTINGS_STORE)) {
          db.createObjectStore(SETTINGS_STORE, {
            keyPath: 'id',
          });
        }
      }

      if (oldVersion < 4) {
        if (!db.objectStoreNames.contains(ZUSTAND_STORE)) {
          // Key-value store for Zustand persist middleware
          db.createObjectStore(ZUSTAND_STORE);
        }
      }
    };
  });

  return dbPromise;
}

export async function saveMessage(message: Message): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(MESSAGES_STORE, 'readwrite');
  const store = tx.objectStore(MESSAGES_STORE);

  const messageToStore = {
    ...message,
    timestamp: message.timestamp.toISOString(),
  };

  await new Promise<void>((resolve, reject) => {
    const request = store.put(messageToStore);
    request.onsuccess = () => resolve();
    request.onerror = () => reject(request.error);
  });
}

export async function loadMessages(conversationId?: string): Promise<Message[]> {
  try {
    const db = await openDB();
    const tx = db.transaction(MESSAGES_STORE, 'readonly');
    const store = tx.objectStore(MESSAGES_STORE);

    let messages: Message[];

    if (conversationId) {
      const index = store.index('conversationId');
      const results = await new Promise<Message[]>((resolve, reject) => {
        const request = index.getAll(conversationId);
        request.onsuccess = () => resolve(request.result as Message[]);
        request.onerror = () => reject(request.error);
      });
      messages = results;
    } else {
      const results = await new Promise<Message[]>((resolve, reject) => {
        const request = store.getAll();
        request.onsuccess = () => resolve(request.result as Message[]);
        request.onerror = () => reject(request.error);
      });
      messages = results;
    }

    return messages
      .map((msg) => ({
        ...msg,
        timestamp: new Date(msg.timestamp),
      }))
      .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
  } catch (error) {
    console.error('Failed to load messages from IndexedDB:', error);
    return [];
  }
}

export async function clearMessages(): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(MESSAGES_STORE, 'readwrite');
  const store = tx.objectStore(MESSAGES_STORE);

  await new Promise<void>((resolve, reject) => {
    const request = store.clear();
    request.onsuccess = () => resolve();
    request.onerror = () => reject(request.error);
  });
}

export async function deleteMessage(id: string): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(MESSAGES_STORE, 'readwrite');
  const store = tx.objectStore(MESSAGES_STORE);

  await new Promise<void>((resolve, reject) => {
    const request = store.delete(id);
    request.onsuccess = () => resolve();
    request.onerror = () => reject(request.error);
  });
}

export async function saveConversation(conversation: Conversation): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(CONVERSATIONS_STORE, 'readwrite');
  const store = tx.objectStore(CONVERSATIONS_STORE);

  const conversationToStore = {
    ...conversation,
    createdAt: conversation.createdAt.toISOString(),
    updatedAt: conversation.updatedAt.toISOString(),
  };

  await new Promise<void>((resolve, reject) => {
    const request = store.put(conversationToStore);
    request.onsuccess = () => resolve();
    request.onerror = () => reject(request.error);
  });
}

export async function loadConversations(): Promise<Conversation[]> {
  try {
    const db = await openDB();
    const tx = db.transaction(CONVERSATIONS_STORE, 'readonly');
    const store = tx.objectStore(CONVERSATIONS_STORE);

    const conversations = await new Promise<Conversation[]>((resolve, reject) => {
      const request = store.getAll();
      request.onsuccess = () => resolve(request.result as Conversation[]);
      request.onerror = () => reject(request.error);
    });

    return conversations
      .map((conv) => ({
        ...conv,
        createdAt: new Date(conv.createdAt),
        updatedAt: new Date(conv.updatedAt),
      }))
      .sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime());
  } catch (error) {
    console.error('Failed to load conversations from IndexedDB:', error);
    return [];
  }
}

export async function deleteConversation(id: string): Promise<void> {
  const db = await openDB();
  const tx = db.transaction([CONVERSATIONS_STORE, MESSAGES_STORE], 'readwrite');
  const conversationStore = tx.objectStore(CONVERSATIONS_STORE);
  const messageStore = tx.objectStore(MESSAGES_STORE);

  await new Promise<void>((resolve, reject) => {
    const deleteConv = conversationStore.delete(id);
    deleteConv.onerror = () => reject(deleteConv.error);

    const index = messageStore.index('conversationId');
    const messagesRequest = index.getAllKeys(id);

    messagesRequest.onsuccess = () => {
      const keys = messagesRequest.result;
      let deleted = 0;

      if (keys.length === 0) {
        resolve();
        return;
      }

      for (const key of keys) {
        const deleteReq = messageStore.delete(key);
        deleteReq.onsuccess = () => {
          deleted++;
          if (deleted === keys.length) {
            resolve();
          }
        };
        deleteReq.onerror = () => reject(deleteReq.error);
      }
    };

    messagesRequest.onerror = () => reject(messagesRequest.error);
  });
}

export async function clearAllData(): Promise<void> {
  const db = await openDB();
  const tx = db.transaction([CONVERSATIONS_STORE, MESSAGES_STORE], 'readwrite');
  const conversationStore = tx.objectStore(CONVERSATIONS_STORE);
  const messageStore = tx.objectStore(MESSAGES_STORE);

  await Promise.all([
    new Promise<void>((resolve, reject) => {
      const request = conversationStore.clear();
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    }),
    new Promise<void>((resolve, reject) => {
      const request = messageStore.clear();
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    }),
  ]);
}

export async function saveSettings(settings: Settings): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(SETTINGS_STORE, 'readwrite');
  const store = tx.objectStore(SETTINGS_STORE);

  const settingsToStore = {
    id: 'user-settings',
    ...settings,
  };

  await new Promise<void>((resolve, reject) => {
    const request = store.put(settingsToStore);
    request.onsuccess = () => resolve();
    request.onerror = () => reject(request.error);
  });
}

export async function loadSettings(): Promise<Settings | null> {
  try {
    const db = await openDB();
    const tx = db.transaction(SETTINGS_STORE, 'readonly');
    const store = tx.objectStore(SETTINGS_STORE);

    const settings = await new Promise<Settings | undefined>((resolve, reject) => {
      const request = store.get('user-settings');
      request.onsuccess = () => resolve(request.result as Settings | undefined);
      request.onerror = () => reject(request.error);
    });

    return settings || null;
  } catch (error) {
    console.error('Failed to load settings from IndexedDB:', error);
    return null;
  }
}

// Zustand persist storage adapter
export const zustandStorage = {
  getItem: async (name: string): Promise<string | null> => {
    try {
      const db = await openDB();
      const tx = db.transaction(ZUSTAND_STORE, 'readonly');
      const store = tx.objectStore(ZUSTAND_STORE);

      return new Promise((resolve, reject) => {
        const request = store.get(name);
        request.onsuccess = () => resolve(request.result || null);
        request.onerror = () => reject(request.error);
      });
    } catch (error) {
      console.error('Error getting Zustand state from IndexedDB:', error);
      return null;
    }
  },

  setItem: async (name: string, value: string): Promise<void> => {
    try {
      const db = await openDB();
      const tx = db.transaction(ZUSTAND_STORE, 'readwrite');
      const store = tx.objectStore(ZUSTAND_STORE);

      return new Promise((resolve, reject) => {
        const request = store.put(value, name);
        request.onsuccess = () => resolve();
        request.onerror = () => reject(request.error);
      });
    } catch (error) {
      console.error('Error setting Zustand state in IndexedDB:', error);
    }
  },

  removeItem: async (name: string): Promise<void> => {
    try {
      const db = await openDB();
      const tx = db.transaction(ZUSTAND_STORE, 'readwrite');
      const store = tx.objectStore(ZUSTAND_STORE);

      return new Promise((resolve, reject) => {
        const request = store.delete(name);
        request.onsuccess = () => resolve();
        request.onerror = () => reject(request.error);
      });
    } catch (error) {
      console.error('Error removing Zustand state from IndexedDB:', error);
    }
  },
};
