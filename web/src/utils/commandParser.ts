/**
 * Command Parser for Virga Terminal
 *
 * Parses user input and determines command type:
 * - Virga global commands (sessions, beacons, etc.)
 * - Virga session commands (exec, shell, sysinfo, etc.)
 * - Shell commands (passed through to OS shell)
 */

export type CommandType = 'virga-global' | 'virga-session' | 'shell';

export interface ParsedCommand {
  type: CommandType;
  command: string;
  args: string[];
  raw: string;
}

/**
 * Virga global commands - available from main CLI prompt
 */
const VIRGA_GLOBAL_COMMANDS = [
  'help',
  'sessions',
  'beacons',
  'listeners',
  'generate',
  'llama',
  'memdb',
  'workflow',
] as const;

/**
 * Virga session commands - available when interacting with a session
 * These are native Virga commands that don't require 'exec' prefix
 */
const VIRGA_SESSION_COMMANDS = [
  // Session management
  'use',
  'interact',

  // File operations (native Virga commands)
  'cd',
  'ls',
  'pwd',
  'upload',
  'download',

  // Information gathering (native Virga commands)
  'info',
  'netinfo',
  'sysinfo',
  'ps',
  'netstat',
  'log',

  // Port forwarding
  'portfwd',

  // Process management
  'kill',

  // Shell mode (special - enters interactive shell)
  'shell',
] as const;

/**
 * Parse command input and determine its type
 *
 * @param input - Raw command input from user
 * @returns Parsed command object with type and arguments
 */
export function parseCommand(input: string): ParsedCommand {
  const trimmed = input.trim();

  if (!trimmed) {
    return {
      type: 'shell',
      command: '',
      args: [],
      raw: input,
    };
  }

  // Split command and arguments
  // Handle quoted arguments properly
  const parts = splitCommandLine(trimmed);
  const cmd = parts[0].toLowerCase();
  const args = parts.slice(1);

  // Check if Virga global command
  if (isVirgaGlobalCommand(cmd)) {
    return {
      type: 'virga-global',
      command: cmd,
      args,
      raw: trimmed,
    };
  }

  // Check if Virga session command
  if (isVirgaSessionCommand(cmd)) {
    return {
      type: 'virga-session',
      command: cmd,
      args,
      raw: trimmed,
    };
  }

  // In session context, anything not a Virga command is executed via 'exec'
  // This matches Virga CLI behavior where you can run OS commands directly
  return {
    type: 'shell',
    command: 'exec',
    args: [trimmed],
    raw: trimmed,
  };
}

/**
 * Check if command is a Virga global command
 */
export function isVirgaGlobalCommand(cmd: string): boolean {
  return VIRGA_GLOBAL_COMMANDS.includes(cmd as any);
}

/**
 * Check if command is a Virga session command
 */
export function isVirgaSessionCommand(cmd: string): boolean {
  return VIRGA_SESSION_COMMANDS.includes(cmd as any);
}

/**
 * Check if command is any Virga command (global or session)
 */
export function isVirgaCommand(cmd: string): boolean {
  return isVirgaGlobalCommand(cmd) || isVirgaSessionCommand(cmd);
}

/**
 * Split command line into parts, respecting quoted arguments
 *
 * Example:
 *   'exec "echo hello world"' -> ['exec', 'echo hello world']
 *   'upload file.txt /tmp/file.txt' -> ['upload', 'file.txt', '/tmp/file.txt']
 */
function splitCommandLine(input: string): string[] {
  const parts: string[] = [];
  let current = '';
  let inQuote = false;
  let quoteChar = '';

  for (let i = 0; i < input.length; i++) {
    const char = input[i];

    if ((char === '"' || char === "'") && !inQuote) {
      // Start of quoted section
      inQuote = true;
      quoteChar = char;
    } else if (char === quoteChar && inQuote) {
      // End of quoted section
      inQuote = false;
      quoteChar = '';
    } else if (char === ' ' && !inQuote) {
      // Space outside quotes - word boundary
      if (current) {
        parts.push(current);
        current = '';
      }
    } else {
      // Regular character
      current += char;
    }
  }

  // Add last part
  if (current) {
    parts.push(current);
  }

  return parts;
}

/**
 * Get help text for a Virga command
 */
export function getCommandHelp(cmd: string): string | null {
  const helpTexts: Record<string, string> = {
    help: 'Display help information for commands\nUsage: help [command]',
    sessions: 'Manage sessions\nUsage: sessions [list|info|kill]',
    beacons: 'Manage beacons\nUsage: beacons [list|generate|configure]',
    listeners: 'Manage listeners\nUsage: listeners [list|start|stop]',
    generate: 'Generate a new beacon\nUsage: generate beacon [options]',
    llama: 'AI features and Llama model management\nUsage: llama [query|config]',
    memdb: 'Memory database operations\nUsage: memdb [list|query]',
    workflow: 'Workflow management\nUsage: workflow [list|run|create]',

    use: 'Select a session to interact with\nUsage: use <session-id>',
    exec: 'Execute a command on the target\nUsage: exec <command>',
    shell: 'Execute a shell command\nUsage: shell <command>',
    sysinfo: 'Get system information\nUsage: sysinfo',
    ps: 'List running processes\nUsage: ps',
    netstat: 'Show network connections\nUsage: netstat',
    portfwd: 'Configure port forwarding\nUsage: portfwd [options]',
    upload: 'Upload a file to target\nUsage: upload <local-path> <remote-path>',
    download: 'Download a file from target\nUsage: download <remote-path> <local-path>',
    cd: 'Change directory on target\nUsage: cd <path>',
    ls: 'List directory contents\nUsage: ls [path]',
    pwd: 'Print working directory\nUsage: pwd',
    info: 'Get session information\nUsage: info',
    netinfo: 'Get network information\nUsage: netinfo',
    log: 'View session logs\nUsage: log [options]',
    interact: 'Start interactive session\nUsage: interact',
    kill: 'Terminate a process\nUsage: kill <pid>',
  };

  return helpTexts[cmd.toLowerCase()] || null;
}

/**
 * Get all available Virga commands
 */
export function getAllVirgaCommands(): string[] {
  return [...VIRGA_GLOBAL_COMMANDS, ...VIRGA_SESSION_COMMANDS];
}

/**
 * Validate command in session context
 * Some commands only make sense in certain contexts
 */
export function validateCommandContext(
  parsed: ParsedCommand,
  hasActiveSession: boolean
): { valid: boolean; error?: string } {
  if (parsed.type === 'virga-session' && !hasActiveSession) {
    return {
      valid: false,
      error: `Command '${parsed.command}' requires an active session. Use 'sessions list' to see available sessions.`,
    };
  }

  return { valid: true };
}
