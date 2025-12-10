/**
 * Extensions API
 *
 * API endpoints for extension management (Windows DLL extensions)
 */

import type { ApiClient } from '../apiClient';
import type {
  BackendExtension,
  BackendExtensionsResponse,
  BackendUploadExtensionResponse,
  BackendExecuteExtensionResponse,
  BackendDeleteExtensionResponse,
} from '../../types/backend';

export class ExtensionsApi {
  constructor(private client: ApiClient) {}

  /**
   * Get extension list
   * GET /api/sessions/{id}/extensions
   */
  async getExtensions(sessionId: string): Promise<BackendExtension[]> {
    const response = await this.client.get<BackendExtensionsResponse>(
      `/api/sessions/${sessionId}/extensions`
    );
    return response.extensions;
  }

  /**
   * Upload extension
   * POST /api/sessions/{id}/extensions
   */
  async uploadExtension(
    sessionId: string,
    file: File,
    onProgress?: (progress: number) => void
  ): Promise<void> {
    const formData = new FormData();
    formData.append('file', file);

    await this.client.upload<BackendUploadExtensionResponse>(
      `/api/sessions/${sessionId}/extensions`,
      formData,
      (progressEvent: { loaded: number; total?: number }) => {
        if (onProgress && progressEvent.total) {
          const percentCompleted = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          );
          onProgress(percentCompleted);
        }
      }
    );
  }

  /**
   * Execute extension
   * POST /api/sessions/{id}/extensions/{name}/execute
   */
  async executeExtension(
    sessionId: string,
    extensionName: string,
    functionName: string,
    args?: string[]
  ): Promise<string> {
    const response = await this.client.post<BackendExecuteExtensionResponse>(
      `/api/sessions/${sessionId}/extensions/${extensionName}/execute`,
      {
        function: functionName,
        arguments: args || [],
      }
    );
    return response.task_id;
  }

  /**
   * Delete extension
   * DELETE /api/sessions/{id}/extensions/{name}
   */
  async deleteExtension(sessionId: string, extensionName: string): Promise<void> {
    await this.client.delete<BackendDeleteExtensionResponse>(
      `/api/sessions/${sessionId}/extensions/${extensionName}`
    );
  }

  /**
   * Get specific extension details
   */
  async getExtension(
    sessionId: string,
    extensionName: string
  ): Promise<BackendExtension | null> {
    const extensions = await this.getExtensions(sessionId);
    return extensions.find((e) => e.name === extensionName) || null;
  }
}
