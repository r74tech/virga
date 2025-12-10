# Virga

A modern Command & Control (C2) framework with local LLM embedded in beacon implants for autonomous post-exploitation. By running AI entirely on the compromised host, Virga minimizes C2 connections and enables agents to complete complex tasks independently with natural language instructions.

## Legal Notice

**WARNING**: This tool is intended exclusively for authorized security testing and research. Use of this software for attacking targets without prior mutual consent is illegal. It is the end user's responsibility to obey all applicable local, state, and federal laws. The developers assume no liability and are not responsible for any misuse or damage caused by this program.

## Quick Start

```bash
# Clone the repository
git clone https://github.com/r74tech/virga.git
cd virga

# Build everything
make

# Run server and CLI together
make run

# Or run separately
make run-server  # Terminal 1
make run-cli     # Terminal 2
```

## Requirements

- Go 1.24+
- gcc
- Make
- mingw (for Windows cross-compilation)
- bun (for web UI development, documentation)

## Installation

### Building from Source
```bash
# Build and run without installation
make deps
make download-llama-all

# Build all components
make build
```

## Architecture

```
virga/
├── cmd/                   # Entry points
│   ├── server/            # C2 server
│   ├── cli/               # Command-line interface
│   └── mcp-stdio/         # MCP STDIO server
├── internal/              # Core implementation
│   ├── server/            # Server logic
│   ├── cli/               # CLI implementation
│   ├── implant/           # Implant/agent code
│   └── shared/            # Shared utilities
├── web/                   # Web dashboard (React + Vite)
├── docs/                  # Documentation (VitePress)
├── examples/              # Extension examples
├── configs/               # Configuration files
└── scripts/               # Build and utility scripts
```

## Core Features

### Server
- Multi-protocol support (HTTP/HTTPS with AES-256-GCM encryption)
- Session management with real-time monitoring
- File transfer and remote command execution
- TLS support with custom certificates

### CLI
- Interactive shell with command completion
- Direct command execution mode (--exec)
- Beacon/payload generation
- Session interaction and management
- File upload/download operations
- Configurable output (quiet mode, color control)

### Web Dashboard
- Real-time session monitoring and management
- Interactive command execution interface
- Network topology visualization with Cytoscape

### Implant/Beacon
- Cross-platform support (Windows, Linux, macOS on x64/ARM64)
- Embedded llama AI for autonomous operations (see [AI Features](docs/usage/ai-features.md))
- In-memory database (MemDB) for data collection
- Configurable sleep intervals with jitter
- Environment detection and anti-debugging

## Common Tasks

### Building
```bash
make                    # Build all components
make server            # Build server only
make cli               # Build CLI only
make clean             # Clean build artifacts
```

### Running
```bash
make run               # Run server and CLI together
make run-server        # Run server only
make run-cli           # Run CLI only
make run-local         # Run without installation
make kill-ports        # Kill processes on default ports
```

### Development
```bash
make dev               # Hot reload mode (requires entr)
```

### Web UI Development
```bash
cd web
bun install            # Install dependencies
bun run dev            # Start development server
bun run build          # Build for production
```

### Documentation
```bash
cd docs
bun install            # Install dependencies
bun run docs:dev       # Start documentation server
bun run docs:build     # Build documentation
```

### Installation
```bash
make install           # Install to system (may need sudo)
make install-user      # Install to ~/.local/bin
make uninstall         # Remove installation
```

## Configuration

### Server Configuration

For example configuration, see `configs/server.yaml`.

### Environment Variables
```bash
# Server
SERVER_PORT=9000 make run-server
SERVER_CONFIG=my-config.yaml make run-server

# CLI
CLI_HOST=192.168.1.100 make run-cli
CLI_PORT=9000 make run-cli
CLI_ARGS="--log-level off --no-color" make run-cli

# Installation
INSTALL_DIR=/custom/path make install
```

### CLI Options
```bash
# Output control
virga-cli --quiet                    # Minimal output
virga-cli --no-color                 # Disable colors
virga-cli --log-level off            # Disable logging

# Debug mode
virga-cli --debug                    # Enable debug output
VIRGA_CLI_LOG_LEVEL=debug virga-cli  # Via environment variable

# Connection options
virga-cli --host 192.168.1.100 --port 9000
virga-cli --config custom-config.yaml
```

## MCP Inspector

For MCP development and debugging (optional):

```bash
# Run MCP inspector (uses bunx, no installation needed)
bun mcp:inspect
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes using conventional commits
4. Open a Pull Request

## Security

- All communications use TLS encryption
- Authentication tokens are never stored on disk
- Input validation on all user inputs
- Regular security audits with Trivy and Gosec

For security issues, please email r74tech@proton.me instead of using issue tracker.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Qitmeer/llama.go](https://github.com/Qitmeer/llama.go) for Go bindings to llama.cpp
- Cobalt Strike for BOF format inspiration
- The Go community for excellent libraries and tools
- Security researchers for valuable feedback and contributions
