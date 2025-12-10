/**
 * Command Router for Virga Terminal
 *
 * Routes commands to appropriate handlers based on command type:
 * - Virga commands -> Virga CLI/API
 * - Shell commands -> Shell execution via API
 */

import type { Session } from '../types/session';
import type { ParsedCommand } from '../utils/commandParser';
import { apiClient } from './apiClient';
import { config } from '../config/env';

export interface CommandOutput {
  content: string;
  isError: boolean;
  type: 'text' | 'table' | 'json';
}

// API endpoint URL
const API_BASE = config.apiBaseURL;

/**
 * Execute a parsed command
 *
 * @param parsed - Parsed command object
 * @param session - Current session context
 * @returns Command output
 */
export async function executeCommand(
  parsed: ParsedCommand,
  session: Session
): Promise<CommandOutput> {
  switch (parsed.type) {
    case 'virga-global':
      return executeVirgaGlobalCommand(parsed, session);

    case 'virga-session':
      return executeVirgaSessionCommand(parsed, session);

    case 'shell':
      return executeShellCommand(parsed, session);

    default:
      return {
        content: `Unknown command type: ${parsed.type}`,
        isError: true,
        type: 'text',
      };
  }
}

/**
 * Execute Virga global command
 */
async function executeVirgaGlobalCommand(
  parsed: ParsedCommand,
  _session: Session
): Promise<CommandOutput> {
  // Currently showing help for most Virga global commands
  // These would normally interact with Virga server API for listeners, sessions, etc.

  switch (parsed.command) {
    case 'help':
      if (parsed.args.length > 0) {
        const { getCommandHelp } = await import('../utils/commandParser');
        const helpText = getCommandHelp(parsed.args[0]);
        if (helpText) {
          return {
            content: helpText,
            isError: false,
            type: 'text',
          };
        }
        return {
          content: `No help available for command: ${parsed.args[0]}`,
          isError: true,
          type: 'text',
        };
      }

      return {
        content: `Available Virga Commands:

Global Commands:
  help           - Display help information
  sessions       - Manage sessions (use Web UI Sessions page)
  beacons        - Manage beacons (use Web UI)
  listeners      - Manage listeners (use Web UI Listeners page)
  generate       - Generate beacons (use Web UI)

Session Commands (require active session):
  exec           - Execute command on target
  shell          - Run shell command on target
  sysinfo        - System information
  ps             - Process list
  netstat        - Network connections
  portfwd        - Port forwarding
  upload/download - File transfer
  cd/ls/pwd      - File navigation

Type 'help <command>' for detailed information.
Type any shell command to execute it on the target system.`,
        isError: false,
        type: 'text',
      };

    default:
      return {
        content: `Virga command '${parsed.command}' - Please use the Web UI for this functionality.\nFor target system commands, type them directly (they will be executed via 'shell').`,
        isError: false,
        type: 'text',
      };
  }
}

/**
 * Execute Virga session command
 */
async function executeVirgaSessionCommand(
  parsed: ParsedCommand,
  session: Session
): Promise<CommandOutput> {
  // Most session commands are executed via shell
  // Some commands require special handling

  switch (parsed.command) {
    case 'sysinfo':
      return executeActualTask(session.id, 'sysinfo', {});

    case 'ps':
      return executeActualTask(session.id, 'ps', {});

    case 'netstat':
      return executeActualTask(session.id, 'netstat', {});

    case 'pwd':
      return executeActualTask(session.id, 'shell', { command: 'pwd' });

    case 'ls':
      return executeActualTask(session.id, 'shell', { command: parsed.args.length > 0 ? `ls ${parsed.args.join(' ')}` : 'ls' });

    case 'cd':
      if (parsed.args.length === 0) {
        return {
          content: 'Usage: cd <directory>',
          isError: true,
          type: 'text',
        };
      }
      return executeActualTask(session.id, 'shell', { command: `cd ${parsed.args.join(' ')} && pwd` });

    default:
      return {
        content: `Session command '${parsed.command}' not yet implemented.\nTry typing shell commands directly.`,
        isError: false,
        type: 'text',
      };
  }
}

/**
 * Execute shell command via Virga API
 */
async function executeShellCommand(
  parsed: ParsedCommand,
  session: Session
): Promise<CommandOutput> {
  const command = parsed.args[0] || parsed.raw;
  return executeActualTask(session.id, 'shell', { command });
}

/**
 * Execute a task via the backend API and wait for results
 */
async function executeActualTask(
  sessionId: string,
  taskType: string,
  payload: Record<string, unknown>
): Promise<CommandOutput> {
  try {
    // Add task to queue
    const addResponse = await apiClient.post<{ task_id: string; success: boolean; message: string }>(
      `${API_BASE}/api/sessions/${sessionId}/tasks`,
      {
        type: taskType,
        payload,
      }
    );

    if (!addResponse.success) {
      return {
        content: `Failed to queue task: ${addResponse.message}`,
        isError: true,
        type: 'text',
      };
    }

    const taskId = addResponse.task_id;

    // Poll for result (max 120 seconds, polling every 2 seconds)
    const maxAttempts = 60;
    const pollInterval = 2000; // 2 seconds

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      // Wait before polling
      await new Promise(resolve => setTimeout(resolve, pollInterval));

      // Fetch specific task result by ID
      const result = await apiClient.get<{
        success: boolean;
        task_id: string;
        output: string;
        exit_code: number;
        error?: string;
        time: string;
        is_complete: boolean;
      }>(`${API_BASE}/api/sessions/${sessionId}/tasks/${taskId}`);

      if (!result.success) {
        continue;
      }

      console.log('[CommandRouter] Poll result:', { taskId, result });

      // Check if the task is complete
      if (result.is_complete) {
        if (result.exit_code === 0) {
          return {
            content: result.output || '(no output)',
            isError: false,
            type: 'text',
          };
        }
        const errorMsg = result.error || result.output || 'Command failed';
        return {
          content: errorMsg,
          isError: true,
          type: 'text',
        };
      }
    }

    // Task timed out
    const totalSeconds = (maxAttempts * pollInterval) / 1000;
    return {
      content: `Task ${taskId} timed out after ${totalSeconds} seconds`,
      isError: true,
      type: 'text',
    };

  } catch (error) {
    if (error instanceof Error) {
      return {
        content: `Error: ${error.message}`,
        isError: true,
        type: 'text',
      };
    }
    return {
      content: 'Unknown error occurred',
      isError: true,
      type: 'text',
    };
  }
}
