# Troubleshooting Guide

This guide provides solutions to common problems you might encounter while using Virga.

## 1. Server Startup & Configuration Issues

Problems related to starting the `virga-server`.

### Fatal error: config validation failed

This is the most common startup error. It means your `config.yaml` file has an issue. The server performs several checks upon loading the configuration.

- **`database path is required`**: Your `config.yaml` is missing the `database.path` field. Add it under the `database` section.
  ```yaml
  database:
    path: "data/virga.db"
    type: "sqlite3"
  ```

- **`at least one listener is required`**: You must define at least one listener under the `listeners:` section.

- **`listener[0]: name is required`**: Every listener in your list must have a `name` field.

- **`listener[0]: invalid port number`**: The `port` for a listener must be a valid number (e.g., 1-65535).

- **`listener[0]: SSL enabled but missing cert/key`**: If you set `use_ssl: true` for a listener, you must provide valid paths for both `ssl.cert` and `ssl.key`.

### Address already in use

This error means another process is already using the port specified for a listener or the admin interface.

- **Solution**: Identify and stop the conflicting process, or change the `port` in your `config.yaml` to an unused one.

## 2. Beacon Generation Issues

Problems encountered when using the `generate beacon` command in the CLI.

### LLM model or library not found

This is the most common issue when generating an AI-enabled beacon.

- **Symptom**: The command fails with an error message like `llama library not found` or `llama model not found`.
- **Cause**: The required AI model and platform-specific libraries have not been downloaded.
- **Solution**: Run the following command from the project's root directory on the machine running the Virgaer:
  ```bash
  make download-llama-deps
  ```

### Cross-Compilation Errors (CGO)

- **Symptom**: The build fails with CGO-related errors, especially when generating a beacon for a different operating system (e.g., building a Windows beacon from Linux).
- **Cause**: Generating a beacon with AI features (`--enable-llama`) requires CGO. Cross-compiling CGO applications requires a specific toolchain.
- **Solution**: Install the appropriate cross-compiler on your system. For example, to build for Windows from a Debian/Ubuntu system, you need `mingw-w64`:
  ```bash
  sudo apt install mingw-w64
  ```

## 3. Beacon Runtime & Connectivity Issues

Problems after a beacon has been generated and deployed to a target.

### Beacon does not connect / No session appears

- **Connectivity**: Ensure the target machine can reach the C2 server and port specified during beacon generation. Check firewalls, network ACLs, and routing.
- **Listener Mismatch**: Double-check that a listener is active on the server that matches the protocol, port, and host embedded in the beacon.
- **Encryption Key Mismatch**: The encryption key is set on the *listener* in the server's `config.yaml`. The beacon will be generated to use this key. If you change the key on the server, you must regenerate the beacon.
- **SSL/TLS Errors**: If using an `https` listener, ensure the target machine trusts the SSL certificate used by the server. For testing, self-signed certificates will often cause the connection to fail unless the beacon is specifically configured to ignore certificate errors (a feature not yet implemented).

### AI Features Fail on Target

- **Symptom**: The beacon connects, but `llama` commands fail or the AI does not behave as expected.
- **Cause 1**: The beacon was not built with AI capabilities.
  - **Solution**: Ensure you generated the beacon using the `--enable-llama` flag or with `llama.enabled: true` in your `beacon.yaml`.
- **Cause 2**: Insufficient memory on the target machine.
  - **Solution**: The AI model requires at least 2-4 GB of free RAM to load. The beacon may crash if the target does not have enough memory.

## 4. CLI Usage Issues

### Cannot connect to server

- **Symptom**: The CLI fails to connect, showing a connection refused error.
- **Cause**: You might be trying to connect to a *listener* port instead of the *admin* port.
- **Solution**: Ensure the `--host` and `--port` flags for the CLI point to the `admin_port` defined in the server's `config.yaml` (default: 8443), not a beacon listener port (like 80 or 443).

### "No active session" error

- **Symptom**: Commands like `exec` or `ls` fail with this error.
- **Cause**: These commands operate within the context of a specific session. You have not selected one yet.
- **Solution**: First, list the available sessions with `sessions list`. Then, select one to interact with using `interact <session_id>`.
