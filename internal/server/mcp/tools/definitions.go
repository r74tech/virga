// Package tools provides centralized MCP tool definitions for all transports.
// This ensures consistency across STDIO, SSE, and Streamable implementations.
package tools

import (
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// SessionListTool lists all active C2 sessions
var SessionListTool = &protocol.Tool{
	Name: "session_list",
	Description: `List all active C2 sessions in the Virga C2 framework. Returns a comprehensive overview of all connected agents/beacons.

Output format: Text-based list containing:
- Total active session count
- For each session:
  - ID: Unique session identifier (UUID format, 36 characters)
  - User: Username@Hostname of the compromised system
  - OS: Operating system and architecture (Windows/Linux/Darwin) (x86/x64/arm64)
  - IP: External IP address of the agent
  - Last Seen: Timestamp of last check-in (ISO8601 format)

Sessions automatically timeout after 5 minutes of inactivity.
No parameters required - this is a read-only operation.
Use this as the first command to understand the current operational picture before executing targeted commands.

Example output:
Active sessions: 2

ID: 550e8400-e29b-41d4-a716-446655440000
  User: admin@DESKTOP-ABC123
  OS: Windows (x64)
  IP: 192.168.1.100
  Last Seen: 2024-01-20 15:04:05

ID: 6ba7b810-9dad-11d1-80b4-00c04fd430c8
  User: root@web-server
  OS: Linux (x64)
  IP: 10.0.0.50
  Last Seen: 2024-01-20 15:03:45`,
	InputSchema: protocol.InputSchema{
		Type:       protocol.Object,
		Properties: map[string]*protocol.Property{},
		Required:   []string{},
	},
}

// SessionCommandTool executes commands on a specific C2 session
var SessionCommandTool = &protocol.Tool{
	Name: "session_command",
	Description: `Execute a command on a specific C2 session. This is the primary interface for remote command execution in Virga.

Supports three distinct execution modes:

1. SHELL COMMANDS (default):
   - Executes OS commands directly on the target system
   - Command length limit: 4096 characters
   - Timeout: 30 seconds
   - Examples: "whoami", "ls -la", "ipconfig /all"

2. LLAMA AI MODE (use_llama=true):
   - Interprets natural language prompts using embedded AI
   - Executes commands autonomously based on intent
   - Can perform complex multi-step operations
   - Temperature: 0.0-1.0 (default: 0.3)
     - 0.1-0.3: Deterministic, focused tasks
     - 0.4-0.6: Balanced exploration
     - 0.7-0.9: Creative problem-solving
   - Max iterations: 1-100 (default: 5)
   - Timeout: 5 minutes
   - Examples: "Find all password files", "Check for persistence mechanisms"

3. MEMDB QUERIES (prefix: "memdb "):
   - Queries the agent's in-memory database
   - Uses SQL-like syntax
   - Available tables: processes, connections, files, registry, services
   - Examples: "memdb SELECT * FROM processes WHERE name LIKE '%ssh%'"

Returns:
- STDIO: Task ID for async tracking
- SSE/Streamable: Direct command output

Error conditions:
- Invalid session_id: "session not found"
- Command timeout: Appropriate timeout message
- Execution failure: Error details with exit code`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id", "command"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session identifier. Must be an active session ID obtained from session_list.
Format: UUID (36 characters, lowercase, hyphens)
Example: "550e8400-e29b-41d4-a716-446655440000"
Validation: Must match ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
			},
			"command": {
				Type: protocol.String,
				Description: `Command to execute on the target system.
Maximum length: 4096 characters
Character encoding: UTF-8

For SHELL mode:
- Any valid OS command for the target platform
- Windows examples: "dir", "net user", "systeminfo"
- Linux examples: "ls -la", "id", "ps aux"

For LLAMA mode:
- Natural language prompt describing the desired action
- Examples:
  - "Find all files modified in the last 24 hours"
  - "Identify running security software"
  - "Check network connections to external IPs"

For MEMDB mode:
- Prefix with "memdb " (case insensitive)
- SQL-like query syntax
- Examples:
  - "memdb SELECT * FROM processes"
  - "memdb SELECT pid, name FROM connections WHERE state = 'ESTABLISHED'"`,
			},
			"use_llama": {
				Type: protocol.Boolean,
				Description: `Enable Llama AI mode for natural language command interpretation.
Default: false
When true: Command is treated as a prompt for the AI to interpret and execute autonomously
When false: Command is executed directly as a shell command
Note: Llama mode requires more time (up to 5 minutes) but can handle complex, multi-step tasks`,
			},
			"max_iterations": {
				Type: protocol.Integer,
				Description: `Maximum number of iterations for Llama AI autonomous execution.
Range: 1-100
Default: 5
Recommended values:
- Simple tasks: 1-5
- Medium complexity: 5-15
- Complex investigations: 15-30
- Extensive operations: 30-100
Note: Higher values increase execution time but allow more thorough task completion`,
			},
			"temperature": {
				Type: protocol.Number,
				Description: `Temperature parameter for Llama AI generation (controls randomness/creativity).
Range: 0.0-1.0
Default: 0.3
Recommended values:
- 0.1-0.3: Precise, deterministic tasks (file searches, data extraction)
- 0.4-0.6: Balanced approach (general reconnaissance)
- 0.7-0.9: Creative problem-solving (finding novel attack vectors)
Note: Lower values produce more predictable, focused results`,
			},
		},
	},
}

// FileUploadTool uploads files to remote sessions
var FileUploadTool = &protocol.Tool{
	Name: "upload_file",
	Description: `Upload a file from the controller to a remote C2 session. Supports binary and text files.

File handling:
- Maximum file size: 10MB (13.3MB when Base64 encoded)
- Automatic parent directory creation if needed
- Preserves file permissions where possible
- Supports both absolute and relative paths

Encoding requirements:
- Content MUST be Base64 encoded before sending
- Use standard Base64 encoding (RFC 4648)
- Binary files: Encode raw bytes
- Text files: Encode UTF-8 bytes

Platform considerations:
- Windows paths: Use backslashes (C:\\Users\\Admin\\file.txt)
- Unix paths: Use forward slashes (/home/user/file.txt)
- Path separators are automatically adjusted for the target OS

Security notes:
- Files are written with agent process permissions
- May fail if target path requires elevated privileges
- Consider operational security when choosing file locations

Common use cases:
- Deploying additional tools or scripts
- Uploading configuration files
- Transferring collected data for exfiltration`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id", "remote_path", "content"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session identifier for file upload.
Must be an active session from session_list.
Format: UUID (36 characters)
Example: "550e8400-e29b-41d4-a716-446655440000"`,
			},
			"remote_path": {
				Type: protocol.String,
				Description: `Full path where the file will be saved on the remote system.
Maximum length: 260 characters (Windows), 4096 characters (Unix)

Format requirements:
- Windows: Drive letter required (C:\\path\\to\\file.txt)
- Unix: Absolute paths recommended (/path/to/file.txt)
- Special characters must be properly escaped
- Parent directories will be created with default permissions

Examples:
- Windows: "C:\\Windows\\Temp\\update.exe"
- Linux: "/tmp/script.sh"
- macOS: "/Users/Shared/data.txt"`,
			},
			"content": {
				Type: protocol.String,
				Description: `Base64 encoded file content.
Maximum size: 13.3MB (encodes to 10MB file)
Encoding: Standard Base64 (RFC 4648)

To encode files:
- Command line: base64 < input_file
- Python: base64.b64encode(open('file', 'rb').read())
- JavaScript: btoa(fileContent)

Example for "Hello World" text file:
"SGVsbG8gV29ybGQ="`,
			},
		},
	},
}

// FileDownloadTool downloads files from remote sessions
var FileDownloadTool = &protocol.Tool{
	Name: "download_file",
	Description: `Download a file from a remote C2 session to the controller. Retrieves files as Base64 encoded content.

File handling:
- Maximum downloadable size: 10MB
- File existence check performed before download
- Returns Base64 encoded content for binary safety
- Supports both absolute and relative paths

Output format:
- SSE/Streamable: File content in Base64 encoding
- STDIO: Task ID for async retrieval

Platform considerations:
- Windows: Both forward and backslashes accepted
- Unix: Forward slashes only
- Hidden files supported (e.g., .bashrc)
- Symbolic links followed to destination

Security notes:
- Access limited by agent process permissions
- May fail on protected system files
- Consider detection risk when accessing sensitive files

Common targets:
- Configuration files (/etc/passwd, .ssh/config)
- Browser data (cookies, history)
- Registry hives (SAM, SYSTEM)
- Application logs and databases`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id", "remote_path"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session identifier for file download.
Must be an active session from session_list.
Format: UUID (36 characters)
Validation: Must exist in active sessions`,
			},
			"remote_path": {
				Type: protocol.String,
				Description: `Full path of the file to download from the remote system.
Maximum length: 260 characters (Windows), 4096 characters (Unix)

Format requirements:
- Must be a valid path on the target OS
- File must exist and be readable
- Can use environment variables (e.g., %USERPROFILE%\\Documents)

Examples:
- Windows: "C:\\Windows\\System32\\config\\SAM"
- Linux: "/etc/shadow", "/home/user/.ssh/id_rsa"
- macOS: "/Library/Keychains/System.keychain"`,
			},
		},
	},
}

// ProcessListTool lists running processes
var ProcessListTool = &protocol.Tool{
	Name: "list_processes",
	Description: `List all running processes on a specific C2 session. Provides comprehensive process enumeration for reconnaissance and security analysis.

Output includes for each process:
- PID: Process ID (integer)
- PPID: Parent process ID (integer)
- Name: Executable name (string)
- Path: Full executable path (string)
- User: Process owner (string)
- CPU%: CPU usage percentage (float)
- Memory%: Memory usage percentage (float)
- Threads: Thread count (integer)
- Start Time: Process start timestamp
- Command Line: Full command with arguments

Platform differences:
- Windows: Uses WMI/Win32 APIs
- Linux: Parses /proc filesystem
- macOS: Uses sysctl and ps

Privilege requirements:
- Standard user: Can see own processes
- Administrator/root: Can see all processes
- Some fields may be empty without elevated privileges

Use cases:
- Identifying security software (AV/EDR)
- Finding interesting processes for injection
- Discovering running services and applications
- Detecting suspicious or malicious processes`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session identifier for process enumeration.
Must be an active session from session_list.
Note: Full process details require appropriate privileges on the target system`,
			},
		},
	},
}

// KillProcessTool terminates a process
var KillProcessTool = &protocol.Tool{
	Name: "kill_process",
	Description: `Terminate a process on a specific C2 session by PID. Forcefully stops the target process.

Termination methods:
- Windows: TerminateProcess API (immediate termination)
- Unix/Linux: SIGKILL signal (non-catchable)
- No graceful shutdown - data loss possible

Privilege requirements:
- Can only kill processes owned by the agent user
- Administrator/root required for system processes
- Protected processes cannot be terminated

Safety warnings:
- Killing critical system processes may cause:
  - System instability or crashes
  - Data corruption
  - Service failures
- Avoid killing: csrss.exe, winlogon.exe, services.exe, init, systemd

Common targets:
- Security software processes
- Competing malware
- Logging/monitoring services
- User applications

Error handling:
- Access denied: Insufficient privileges
- Process not found: PID doesn't exist
- Success: No output (process terminated)`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id", "pid"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session identifier where the process is running.
Must be an active session from session_list`,
			},
			"pid": {
				Type: protocol.Integer,
				Description: `Process ID to terminate.
Range: 1-4294967295 (32-bit systems) or higher (64-bit)
Must be a valid PID from list_processes output
Example: 1234

Special PIDs to avoid:
- 0: Kernel/idle process
- 1: Init process (Unix)
- 4: System process (Windows)`,
			},
		},
	},
}

// SystemInfoTool gathers comprehensive system information
var SystemInfoTool = &protocol.Tool{
	Name: "get_system_info",
	Description: `Get comprehensive system information from a specific C2 session. Essential for initial reconnaissance and privilege escalation planning.

Information collected includes:

SYSTEM DETAILS:
- OS name, version, build number, architecture
- System uptime and install date
- Hardware: CPU (model, cores, speed), RAM, disk space
- BIOS/UEFI information

NETWORK CONFIGURATION:
- All network interfaces (name, MAC, IPs)
- DNS servers and search domains
- Routing table and gateways
- Active network shares

USER INFORMATION:
- Current user and privileges
- User groups and permissions
- Logged in users (local and remote)
- User directories and profiles

SECURITY SOFTWARE:
- Installed antivirus/EDR products
- Firewall status and rules
- Windows Defender status
- Security patches and hotfixes

ADDITIONAL DATA:
- Environment variables
- Installed software list
- Running services
- Scheduled tasks
- Startup programs

Execution time: 5-30 seconds depending on system
Privilege impact: More information available with elevated privileges`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session identifier for system information gathering.
Must be an active session from session_list.
Note: Some information requires elevated privileges (Administrator/root)`,
			},
		},
	},
}

// NetworkConnectionsTool lists network connections
var NetworkConnectionsTool = &protocol.Tool{
	Name: "list_network_connections",
	Description: `List all network connections on a specific C2 session. Provides netstat-like output for network reconnaissance.

Information per connection:
- Protocol: TCP/UDP
- Local Address: IP and port
- Remote Address: IP and port (if connected)
- State: Connection state
  - ESTABLISHED: Active connection
  - LISTENING: Waiting for connections
  - TIME_WAIT: Recently closed
  - CLOSE_WAIT: Remote end closed
  - SYN_SENT: Connection attempt
- PID: Process ID owning the connection
- Process Name: Executable name
- Duration: Connection lifetime (if available)

Filtering capabilities:
- All connections shown by default
- Includes both IPv4 and IPv6
- Shows both TCP and UDP

Privilege requirements:
- Standard user: Can see own connections
- Administrator/root: Can see all connections with process info

Use cases:
- Identifying C2 channels and backdoors
- Finding lateral movement opportunities
- Discovering network services
- Detecting data exfiltration
- Mapping internal network architecture

Detection considerations:
- This command may trigger EDR alerts
- Similar to running "netstat -an"`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session identifier for network enumeration.
Must be an active session from session_list.
Full connection details require elevated privileges`,
			},
		},
	},
}

// PortForwardTool sets up port forwarding
var PortForwardTool = &protocol.Tool{
	Name: "port_forward",
	Description: `Set up port forwarding through a C2 session to access internal network resources. Creates a tunnel: controller:local_port -> beacon -> remote_host:remote_port.

How it works:
1. Opens a listener on the controller (local_port)
2. Forwards connections through the C2 channel
3. Beacon connects to remote_host:remote_port
4. Bidirectional data flow established

Configuration limits:
- Local port: 1-65535 (1-1024 requires root/admin)
- Remote port: 1-65535
- Multiple forwards can be active simultaneously
- Each forward uses C2 channel bandwidth

Common forwarding scenarios:
- Web interfaces: 8080 -> 192.168.1.100:80
- RDP access: 13389 -> 10.0.0.50:3389
- SSH pivoting: 2222 -> 172.16.0.10:22
- Database access: 15432 -> 10.0.0.20:5432
- SMB shares: 1445 -> 192.168.1.1:445

Network requirements:
- Beacon must have network route to remote_host
- Firewall rules must permit the connection
- Consider bandwidth limitations of C2 channel

Security notes:
- Forwarded traffic is encrypted within C2 channel
- Local port becomes accessible to anyone on controller
- May trigger network monitoring alerts
- Use non-standard local ports to avoid conflicts`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id", "local_port", "remote_host", "remote_port"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Session to use as pivot point.
Must have network access to the remote_host.
The beacon will make the outbound connection`,
			},
			"local_port": {
				Type: protocol.Integer,
				Description: `Local port to listen on (controller side).
Range: 1-65535
Constraints:
- Must not be already in use
- Ports 1-1024 require elevated privileges
- Recommended: Use high ports (10000-65535)
Examples: 8080, 13389, 15432`,
			},
			"remote_host": {
				Type: protocol.String,
				Description: `Target host to forward connections to.
Format: IP address or hostname
Must be reachable from the beacon's network position
Examples:
- IP: "192.168.1.100", "10.0.0.50"
- Hostname: "internal-server", "database.local"
- Localhost: "127.0.0.1" (for beacon's local services)`,
			},
			"remote_port": {
				Type: protocol.Integer,
				Description: `Target port to forward to.
Range: 1-65535
Common ports:
- 22: SSH
- 80: HTTP
- 443: HTTPS
- 445: SMB
- 1433: MSSQL
- 3306: MySQL
- 3389: RDP
- 5432: PostgreSQL
- 5900: VNC`,
			},
		},
	},
}

// InteractBeaconTool starts interactive mode (Streamable only)
var InteractBeaconTool = &protocol.Tool{
	Name: "interact_beacon",
	Description: `Start interactive session with a beacon for real-time command execution. Switches the beacon from default 30-second check-in to 1-second interval.

Mode changes:
- Check-in interval: 30s -> 1s (30x faster)
- Enables rapid command-response cycles
- Ideal for hands-on operations
- Persistent until explicitly stopped

Use cases:
- Active exploitation and post-exploitation
- Real-time system exploration
- Debugging and troubleshooting
- Time-sensitive operations
- Interactive file/registry manipulation

Network impact:
- Increased traffic (30x more check-ins)
- Higher bandwidth usage
- More visible to network monitoring
- Increased battery usage on mobile devices

Best practices:
- Use for short periods only
- Return to normal mode when done
- Avoid during stealth operations
- Monitor network detection risk

Operational security:
- Frequent beaconing may trigger alerts
- Pattern detection risk increases
- Use stop_interact to return to normal`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session to enable interactive mode.
Session will immediately switch to 1-second check-ins.
Use stop_interact to return to normal 30-second interval`,
			},
		},
	},
}

// StopInteractTool stops interactive mode (Streamable only)
var StopInteractTool = &protocol.Tool{
	Name: "stop_interact",
	Description: `Stop interactive session and return beacon to normal operational mode. Switches check-in interval back from 1-second to 30-second.

Mode changes:
- Check-in interval: 1s -> 30s
- Returns to stealthy operation
- Reduces network traffic
- Decreases detection risk

Always use this when finishing interactive operations to:
- Maintain operational security
- Reduce network footprint
- Conserve endpoint resources
- Avoid pattern detection

The beacon will acknowledge the mode change and resume normal operations immediately.`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Session to disable interactive mode for.
Must be currently in interactive mode.
Returns to 30-second check-in interval`,
			},
		},
	},
}

// ShellTool is a convenient alias for shell commands (Streamable only)
var ShellTool = &protocol.Tool{
	Name: "shell",
	Description: `Execute shell command on a beacon (convenience alias for session_command). Direct shortcut for rapid command execution without specifying type.

This is equivalent to calling session_command with type='shell'.
Ideal for quick operations during interactive sessions.

Command execution:
- Direct OS command execution
- No interpretation or modification
- Platform-specific shell:
  - Windows: cmd.exe
  - Linux/macOS: /bin/sh
- Timeout: 30 seconds

Examples:
- Windows: "dir", "ipconfig", "net user"
- Linux: "ls -la", "id", "ps aux"
- macOS: "sw_vers", "dscl . list /Users"`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id", "command"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session for command execution.
Must be an active session from session_list`,
			},
			"command": {
				Type: protocol.String,
				Description: `Shell command to execute.
Maximum length: 4096 characters.
Will be executed directly without modification`,
			},
		},
	},
}

// LsTool lists directory contents (Streamable only)
var LsTool = &protocol.Tool{
	Name: "ls",
	Description: `List directory contents on a beacon (convenience alias). Executes 'ls -la' with optional path parameter.

Provides detailed file listing including:
- Permissions (rwx format)
- Owner and group
- File size
- Modification timestamp
- Hidden files (dotfiles)

Path handling:
- Default: Current working directory
- Relative paths: Resolved from CWD
- Absolute paths: Direct listing
- Special paths: "~" expands to user home

Platform behavior:
- Linux/macOS: Native ls -la command
- Windows: May require Unix tools or use dir equivalent

Output format:
- Standard ls -la format
- Hidden files included (. prefix)
- Sorted alphabetically`,
	InputSchema: protocol.InputSchema{
		Type:     protocol.Object,
		Required: []string{"session_id"},
		Properties: map[string]*protocol.Property{
			"session_id": {
				Type: protocol.String,
				Description: `Target session for directory listing.
Must be an active session from session_list`,
			},
			"path": {
				Type: protocol.String,
				Description: `Directory path to list.
Optional - defaults to current directory (.)
Examples:
- ".": Current directory
- "/tmp": Absolute path
- "../": Parent directory
- "~/Documents": User's documents`,
			},
		},
	},
}

// GetSystemStatusTool returns Virga server status (Streamable only)
var GetSystemStatusTool = &protocol.Tool{
	Name: "get_system_status",
	Description: `Get comprehensive Virga C2 server system status. Returns server-side statistics and health information.

Information includes:

SERVER INFO:
- Name: Server identifier
- Version: Virga version
- Uptime: Time since server start

SESSION STATISTICS:
- Active: Currently connected sessions
- Total: All sessions in database

BEACON STATISTICS:
- Total: Number of configured beacons

Output format: JSON structure
Use this to monitor C2 infrastructure health and capacity.

Note: This is different from get_system_info which queries target systems.`,
	InputSchema: protocol.InputSchema{
		Type:       protocol.Object,
		Properties: map[string]*protocol.Property{},
		Required:   []string{},
	},
}
