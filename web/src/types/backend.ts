/**
 * Backend Types
 *
 * Virga beacon-server response type definitions
 * Different from the frontend types (session.ts, graph.ts, etc.)
 */

/**
 * Backend session type
 */
export interface BackendSession {
  id: string;
  remote_addr: string; // "192.168.1.10:12345"
  hostname: string;
  username: string;
  os: string;
  last_activity: string; // ISO 8601
  interval?: number; // Beacon interval in seconds
  jitter?: number; // Jitter percentage
  environment: {
    arch: string;
    process_id: number;
    working_dir?: string;
  };
}

/**
 * Session list response
 */
export interface BackendSessionsResponse {
  success: true;
  sessions: BackendSession[];
}

/**
 * Backend task result type (CLI compatible format)
 * GET response for /api/sessions/{id}/tasks/{taskID}
 */
export interface BackendTaskResult {
  success: boolean;
  task_id: string;
  output: string;
  exit_code: number;
  error: string;
  time: string; // ISO 8601
  is_complete: boolean;

  // Web UI extended fields (optional)
  status?: 'pending' | 'running' | 'completed' | 'failed';
  partial?: boolean; // True if this is a partial/intermediate result (in TaskResult array)
}

/**
 * Task progress information (Web UI only)
 * Response for GET /api/sessions/{id}/tasks/{taskID}/progress
 */
export interface TaskProgress {
  current_iteration: number;
  max_iterations: number;
  last_update: number; // Unix timestamp
  iteration_outputs?: string[];
}

/**
 * Task add response
 */
export interface BackendAddTaskResponse {
  success: true;
  task_id: string;
  message: string;
}

/**
 * Task results list response
 */
export interface BackendTaskResultsResponse {
  success: true;
  results: BackendTaskResult[];
}

/**
 * Task progress get response
 */
export interface BackendTaskProgressResponse {
  success: boolean;
  task_id: string;
  progress: TaskProgress | null;
}

/**
 * Interactive mode set response
 */
export interface BackendSetInteractiveResponse {
  success: true;
  message: string;
}

/**
 * Backend listener type
 */
export interface BackendListener {
  name: string;
  type: string;
  bind_address: string;
  port: number;
  status: 'running' | 'stopped' | 'error';
  config?: Record<string, unknown>;
}

/**
 * Listener list response
 */
export interface BackendListenersResponse {
  success: true;
  listeners: BackendListener[];
}

/**
 * Listener create response
 */
export interface BackendCreateListenerResponse {
  success: true;
  message: string;
  listener: BackendListener;
}

/**
 * Listener delete response
 */
export interface BackendDeleteListenerResponse {
  success: true;
  message: string;
}

/**
 * Extension type
 */
export interface BackendExtension {
  name: string;
  version?: string;
  loaded_at: string; // ISO 8601
}

/**
 * Extension list response
 */
export interface BackendExtensionsResponse {
  success: true;
  extensions: BackendExtension[];
}

/**
 * Extension upload response
 */
export interface BackendUploadExtensionResponse {
  success: true;
  message: string;
}

/**
 * Extension execute response
 */
export interface BackendExecuteExtensionResponse {
  success: true;
  task_id: string;
  message: string;
}

/**
 * Extension delete response
 */
export interface BackendDeleteExtensionResponse {
  success: true;
  message: string;
}

/**
 * Generic success response
 */
export interface BackendSuccessResponse {
  success: true;
  message: string;
}
