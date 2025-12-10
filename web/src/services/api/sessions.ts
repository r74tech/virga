/**
 * Sessions API
 *
 * API endpoints for session management and task execution
 */

import type { ApiClient } from '../apiClient';
import type {
  BackendSessionsResponse,
  BackendAddTaskResponse,
  BackendTaskResultsResponse,
  BackendSetInteractiveResponse,
  BackendTaskProgressResponse,
} from '../../types/backend';
import type { Session } from '../../types/session';
import { convertBackendSession } from '../converters';

export class SessionsApi {
  constructor(private client: ApiClient) {}

  /**
   * Get session list
   * GET /api/sessions
   */
  async getSessions(): Promise<Session[]> {
    const response = await this.client.get<BackendSessionsResponse>('/api/sessions');
    return response.sessions.map(convertBackendSession);
  }

  /**
   * Get session details
   * GET /api/sessions/{id}
   *
   * Note: This endpoint is not currently implemented on the server side
   * Instead, use getSessions() to filter the sessions
   */
  async getSession(sessionId: string): Promise<Session | null> {
    // Search for the corresponding session in the list
    const sessions = await this.getSessions();
    return sessions.find((s) => s.id === sessionId) || null;
  }

  /**
   * Add task
   * POST /api/sessions/{id}/tasks
   */
  async addTask(
    sessionId: string,
    type: string,
    payload: Record<string, unknown>
  ): Promise<BackendAddTaskResponse> {
    const response = await this.client.post<BackendAddTaskResponse>(
      `/api/sessions/${sessionId}/tasks`,
      {
        type,
        payload,
      }
    );
    return response;
  }

  /**
   * Get task results list
   * GET /api/sessions/{id}/tasks
   */
  async getTaskResults(sessionId: string) {
    const response = await this.client.get<BackendTaskResultsResponse>(
      `/api/sessions/${sessionId}/tasks`
    );
    return response.results;
  }

  /**
   * Set interactive mode
   * POST /api/sessions/{id}/interactive
   */
  async setInteractiveMode(
    sessionId: string,
    interactive: boolean
  ): Promise<void> {
    await this.client.post<BackendSetInteractiveResponse>(
      `/api/sessions/${sessionId}/interactive`,
      {
        interactive,
      }
    );
  }

  /**
   * Poll task result
   * Check until the task is complete
   */
  async pollTaskResult(
    sessionId: string,
    taskId: string,
    options: {
      interval?: number;
      maxAttempts?: number;
    } = {}
  ) {
    const interval = options.interval || 1000; // 1 second
    const maxAttempts = options.maxAttempts || 60; // 60 seconds

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      const result = await this.getTaskResult(sessionId, taskId);

      if (result.status === 'completed' || result.status === 'failed') {
        return result;
      }

      // Wait for the next attempt
      await new Promise((resolve) => setTimeout(resolve, interval));
    }

    throw new Error(
      `Task ${taskId} did not complete within ${maxAttempts} attempts`
    );
  }

  /**
   * Execute killswitch
   * POST /api/sessions/{id}/tasks (type: killswitch)
   */
  async killswitch(
    sessionId: string,
    options: {
      mode: 'shutdown' | 'remove';
      message?: string;
    }
  ): Promise<BackendAddTaskResponse> {
    return this.addTask(sessionId, 'killswitch', {
      mode: options.mode,
      message: options.message || '',
      confirm: true,
    });
  }

  /**
   * Send chat message
   * POST /api/sessions/{id}/chat
   */
  async sendChatMessage(
    sessionId: string,
    message: string,
    options?: {
      maxIterations?: number;
      temperature?: number;
    }
  ): Promise<{
    taskId: string;
    status: string;
    result?: {
      output: string;
      exitCode: number;
      error: string;
    };
  }> {
    interface ChatResponse {
      success: boolean;
      taskId: string;
      status: string;
      message?: string;
      result?: {
        output: string;
        exitCode: number;
        error: string;
      };
    }

    const response = await this.client.post<ChatResponse>(
      `/api/sessions/${sessionId}/chat`,
      {
        message,
        options: options || {},
      }
    );

    if (!response.success) {
      throw new Error(response.message || 'Failed to send chat message');
    }

    return {
      taskId: response.taskId,
      status: response.status,
      result: response.result,
    };
  }

  /**
   * Get task result
   * GET /api/sessions/{id}/tasks/{taskId}
   */
  async getTaskResult(
    sessionId: string,
    taskId: string
  ): Promise<{
    isComplete: boolean;
    status: string;
    results: Array<{
      taskId: string;
      output: string;
      exitCode: number;
      error?: string;
      time: string;
      partial?: boolean;
    }>;
  }> {
    interface TaskResponse {
      success: boolean;
      task_id: string;
      output: string;
      exit_code: number;
      error: string;
      time: string;
      is_complete: boolean;
      status: string;
      partial_results?: Array<{
        taskId: string;
        output: string;
        exitCode: number;
        error?: string;
        time: string;
        partial?: boolean;
      }>;
    }

    const response = await this.client.get<TaskResponse>(
      `/api/sessions/${sessionId}/tasks/${taskId}`
    );

    // use partial_results if it exists, otherwise create from output
    const results = response.partial_results || [];

    // if output exists and partial_results is empty, create result from output
    if (results.length === 0 && response.output) {
      results.push({
        taskId: response.task_id,
        output: response.output,
        exitCode: response.exit_code,
        error: response.error || undefined,
        time: response.time,
        partial: false,
      });
    }

    return {
      isComplete: response.is_complete,
      status: response.status,
      results,
    };
  }

  /**
   * Poll chat task result (status version)
   * Check until the task is complete
   */
  async pollChatTaskResult(
    sessionId: string,
    taskId: string,
    options: {
      interval?: number;
      maxAttempts?: number;
      onProgress?: (attempt: number) => void;
    } = {}
  ): Promise<{
    isComplete: boolean;
    status: string;
    results: Array<{
      taskId: string;
      output: string;
      exitCode: number;
      error?: string;
      time: string;
      partial?: boolean;
    }>;
  }> {
    const interval = options.interval || 3000; // 3 seconds
    const maxAttempts = options.maxAttempts || 100; // 5 minutes

    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
      if (options.onProgress) {
        options.onProgress(attempt);
      }

      try {
        const result = await this.getTaskResult(sessionId, taskId);

        // status field to determine (Web UI extension)
        if (result.status === 'completed' || result.status === 'failed') {
          return result;
        }

        // is_complete to determine (CLI compatible)
        if (result.isComplete) {
          return result;
        }

        // Still running, waiting
        await new Promise((resolve) => setTimeout(resolve, interval));
      } catch (error) {
        // If 404 error, the task is not found yet, so continue
        if (error instanceof Error && error.message.includes('404')) {
          await new Promise((resolve) => setTimeout(resolve, interval));
          continue;
        }
        // If other errors, rethrow
        throw error;
      }
    }

    throw new Error(
      `Task ${taskId} did not complete within ${(maxAttempts * interval) / 1000} seconds`
    );
  }

  /**
   * Get task progress
   * GET /api/sessions/{id}/tasks/{taskID}/progress
   */
  async getTaskProgress(
    sessionId: string,
    taskId: string
  ): Promise<BackendTaskProgressResponse> {
    const response = await this.client.get<BackendTaskProgressResponse>(
      `/api/sessions/${sessionId}/tasks/${taskId}/progress`
    );
    return response;
  }
}
