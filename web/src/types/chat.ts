export interface Message {
  id: string;
  conversationId: string;
  type: 'user' | 'assistant' | 'system' | 'command';
  content: string;
  timestamp: Date;
  metadata?: MessageMetadata;
}

export interface MessageMetadata {
  command?: string;
  result?: CommandResult;
  sessionId?: string;
  taskId?: string; // Task ID (for progress tracking)
  progress?: TaskProgress; // Task progress information
  status?: string; // Task status
  partialResults?: string[]; // Array of partial results
  isProcessing?: boolean; // Processing flag
}

export interface TaskProgress {
  current_iteration: number;
  max_iterations: number;
  last_update: number;
  iteration_outputs?: string[];
}

export interface CommandResult {
  success: boolean;
  output?: string;
  error?: string;
  duration?: number;
  exitCode?: number;
}

export interface Conversation {
  id: string;
  title: string;
  createdAt: Date;
  updatedAt: Date;
  messageCount: number;
  preview?: string;
}

export interface ChatState {
  conversations: Conversation[];
  currentConversationId: string | null;
  messages: Message[];
  isTyping: boolean;
}
