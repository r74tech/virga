# Makefile for Virga - Command & Control Framework
# Run 'make help' for usage information

.SILENT:
.PHONY: all build clean test help deps
.PHONY: server cli agent mcp-stdio
.PHONY: run run-server run-cli run-mcp-stdio
.PHONY: dev dev-server dev-cli
.PHONY: download-llama-deps download-llama-all
.PHONY: release install uninstall install-user uninstall-user
.PHONY: kill-ports test-coverage deps-update
.PHONY: run-local symlink unsymlink
.PHONY: update-ip generate-key
.PHONY: fmt lint vet

# ==================== Configuration ====================

# Architecture detection
OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)

# Normalize architecture names
ifeq ($(ARCH),x86_64)
	ARCH := amd64
endif
ifeq ($(ARCH),aarch64)
	ARCH := arm64
endif

# Target architecture (can be overridden)
TARGET_OS ?= $(OS)
TARGET_ARCH ?= $(ARCH)
TARGET_PLATFORM := $(TARGET_OS)-$(TARGET_ARCH)

# Version info
VERSION ?= 1.0.0
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%d %H:%M:%S UTC")

# Build configuration
BINARY_PREFIX := virga
BIN_DIR := bin
# Default installation directory (can be overridden)
# Common options: /usr/local/bin, ~/.local/bin, ~/bin, ./bin
INSTALL_DIR ?= $(HOME)/.local/bin

# Server configuration
SERVER_CONFIG ?= configs/server.yaml
SERVER_HOST ?= 127.0.0.1
SERVER_PORT ?= 8443

# CLI configuration
CLI_HOST ?= 127.0.0.1
CLI_PORT ?= 8443
CLI_ARGS ?= --log-level info

# Build flags
LDFLAGS := -X 'github.com/r74tech/virga/internal/cli/ui.Version=$(VERSION)' \
           -X 'github.com/r74tech/virga/internal/cli/ui.GitCommit=$(GIT_COMMIT)' \
           -X 'github.com/r74tech/virga/internal/cli/ui.BuildDate=$(BUILD_DATE)'

# Go build flags
GOFLAGS := -v
GO_BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# ==================== Main Targets ====================

# Default target
all: build

# Build all components
build: server cli mcp-stdio
	@echo "Build complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete!"

# ==================== Build Targets ====================

server:
	@echo "Building server..."
	@mkdir -p $(BIN_DIR)
	@go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_PREFIX)-server ./cmd/server

cli:
	@echo "Building CLI..."
	@mkdir -p $(BIN_DIR)
	@go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_PREFIX)-cli ./cmd/cli

mcp-stdio:
	@echo "Building MCP STDIO server..."
	@mkdir -p $(BIN_DIR)
	@go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_PREFIX)-mcp-stdio ./cmd/mcp-stdio

# ==================== Run Targets ====================

# Run both server and CLI (in background)
run: build
	@echo "Starting Virga server and CLI..."
	@make run-server &
	@sleep 2
	@make run-cli

# Run server with config
run-server: server
	@echo "Starting server on $(SERVER_HOST):$(SERVER_PORT)..."
	@$(BIN_DIR)/$(BINARY_PREFIX)-server --config $(SERVER_CONFIG)

# Run CLI with connection params
run-cli: cli
	@echo "Starting CLI connecting to $(CLI_HOST):$(CLI_PORT)..."
	@$(BIN_DIR)/$(BINARY_PREFIX)-cli --host $(CLI_HOST) --port $(CLI_PORT) $(CLI_ARGS)

# Run MCP STDIO server
run-mcp-stdio: mcp-stdio
	@echo "Starting MCP STDIO server..."
	@$(BIN_DIR)/$(BINARY_PREFIX)-mcp-stdio

# ==================== Development Targets ====================

# Development mode with hot reload (requires entr)
dev: dev-server dev-cli

dev-server:
	@echo "Starting server in development mode..."
	@find . -name "*.go" | entr -r make run-server

dev-cli:
	@echo "Starting CLI in development mode..."
	@find . -name "*.go" | entr -r make run-cli

# ==================== Utility Targets ====================

# Update IP addresses in config files
update-ip:
	@echo "🌐 Updating IP addresses in configuration files..."
	@go run scripts/update-ip.go

# Kill processes on common ports
kill-ports:
	@echo "🔫 Killing processes on ports 8080, 8443, 8444..."
	@for port in 8080 8443 8444; do \
		lsof -ti :$$port | xargs -r kill -9 2>/dev/null || true; \
	done
	@echo "Ports cleared!"

# ==================== Testing ====================

test:
	@echo "Running tests..."
	@go test $(GOFLAGS) ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# ==================== Code Quality ====================

fmt:
	@echo "Formatting Go code..."
	@goimports -w ./cmd ./internal
	@echo "Formatting complete!"

vet:
	@echo "Running go vet..."
	@go vet ./cmd/... ./internal/...

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=5m --tests=false ./cmd/... ./internal/...

# ==================== Dependencies ====================

deps:
	@echo "Updating go.mod for current platform..."
	@go run scripts/download-llama-libs.go -update-go-mod
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy

deps-update:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# ==================== Llama Dependencies ====================

download-llama-deps:
	@echo "Downloading Llama dependencies for $(TARGET_PLATFORM)..."
	@go run scripts/download-llama-libs.go
	@go run scripts/download-llama-model.go

download-llama-all:
	@echo "Downloading all Llama libraries..."
	@go run scripts/download-llama-libs.go -all
	@go run scripts/download-llama-model.go

# ==================== Security Targets ====================

generate-key:
	@echo "Generating secure encryption keys..."
	@scripts/generate-secure-key.sh

# ==================== Release Targets ====================

# Build release binaries for all platforms
release:
	@echo "Building release binaries..."
	@mkdir -p $(BIN_DIR)/release
	# Linux AMD64
	@GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/release/$(BINARY_PREFIX)-server-linux-amd64 ./cmd/server
	@GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/release/$(BINARY_PREFIX)-cli-linux-amd64 ./cmd/cli
	# Darwin AMD64
	@GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/release/$(BINARY_PREFIX)-server-darwin-amd64 ./cmd/server
	@GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/release/$(BINARY_PREFIX)-cli-darwin-amd64 ./cmd/cli
	# Darwin ARM64
	@GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/release/$(BINARY_PREFIX)-server-darwin-arm64 ./cmd/server
	@GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/release/$(BINARY_PREFIX)-cli-darwin-arm64 ./cmd/cli
	# Windows AMD64
	@GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/release/$(BINARY_PREFIX)-server-windows-amd64.exe ./cmd/server
	@GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/release/$(BINARY_PREFIX)-cli-windows-amd64.exe ./cmd/cli
	@echo "Release builds complete!"

# ==================== Local Development ====================

# Run from current directory without installation
run-local: build
	@echo "Running Virga from local build directory..."
	@echo "Add this to your shell profile to use virga commands:"
	@echo "   export PATH=\"$(PWD)/$(BIN_DIR):$$PATH\""
	@echo ""
	@./$(BIN_DIR)/$(BINARY_PREFIX)-cli

# Create symlinks for development (no copy needed)
symlink: build
	@echo "Creating symlinks in $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@ln -sf $(PWD)/$(BIN_DIR)/$(BINARY_PREFIX)-server $(INSTALL_DIR)/$(BINARY_PREFIX)-server
	@ln -sf $(PWD)/$(BIN_DIR)/$(BINARY_PREFIX)-cli $(INSTALL_DIR)/$(BINARY_PREFIX)-cli
	@echo "Symlinks created!"

# Remove symlinks
unsymlink:
	@echo "Removing symlinks from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_PREFIX)-server
	@rm -f $(INSTALL_DIR)/$(BINARY_PREFIX)-cli
	@echo "Symlinks removed!"

# ==================== Installation ====================

install: build
	@echo "Installing Virga to $(INSTALL_DIR)..."
	@if [ -w "$(INSTALL_DIR)" ]; then \
		cp $(BIN_DIR)/$(BINARY_PREFIX)-server $(INSTALL_DIR)/; \
		cp $(BIN_DIR)/$(BINARY_PREFIX)-cli $(INSTALL_DIR)/; \
		chmod +x $(INSTALL_DIR)/$(BINARY_PREFIX)-server; \
		chmod +x $(INSTALL_DIR)/$(BINARY_PREFIX)-cli; \
	else \
		echo "Need sudo permissions to install to $(INSTALL_DIR)"; \
		sudo cp $(BIN_DIR)/$(BINARY_PREFIX)-server $(INSTALL_DIR)/; \
		sudo cp $(BIN_DIR)/$(BINARY_PREFIX)-cli $(INSTALL_DIR)/; \
		sudo chmod +x $(INSTALL_DIR)/$(BINARY_PREFIX)-server; \
		sudo chmod +x $(INSTALL_DIR)/$(BINARY_PREFIX)-cli; \
	fi
	@echo "Installation complete!"

uninstall:
	@echo "Uninstalling Virga from $(INSTALL_DIR)..."
	@if [ -w "$(INSTALL_DIR)" ]; then \
		rm -f $(INSTALL_DIR)/$(BINARY_PREFIX)-server; \
		rm -f $(INSTALL_DIR)/$(BINARY_PREFIX)-cli; \
	else \
		echo "Need sudo permissions to uninstall from $(INSTALL_DIR)"; \
		sudo rm -f $(INSTALL_DIR)/$(BINARY_PREFIX)-server; \
		sudo rm -f $(INSTALL_DIR)/$(BINARY_PREFIX)-cli; \
	fi
	@echo "Uninstallation complete!"

# User-local installation (no sudo required)
install-user: build
	@echo "Installing Virga to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp $(BIN_DIR)/$(BINARY_PREFIX)-server ~/.local/bin/
	@cp $(BIN_DIR)/$(BINARY_PREFIX)-cli ~/.local/bin/
	@chmod +x ~/.local/bin/$(BINARY_PREFIX)-server
	@chmod +x ~/.local/bin/$(BINARY_PREFIX)-cli
	@echo "Installation complete!"
	@echo "Make sure ~/.local/bin is in your PATH"

uninstall-user:
	@echo "Uninstalling Virga from ~/.local/bin..."
	@rm -f ~/.local/bin/$(BINARY_PREFIX)-server
	@rm -f ~/.local/bin/$(BINARY_PREFIX)-cli
	@echo "Uninstallation complete!"

# ==================== Help ====================

help:
	@echo "Virga - Command & Control Framework"
	@echo ""
	@echo "Quick Start:"
	@echo "  make               - Build all components"
	@echo "  make run           - Run server and CLI together"
	@echo "  make run-local     - Run from build directory (no install)"
	@echo ""
	@echo "Build Targets:"
	@echo "  make server        - Build server only"
	@echo "  make cli           - Build CLI only"
	@echo "  make mcp-stdio     - Build MCP STDIO server"
	@echo "  make clean         - Remove all build artifacts"
	@echo ""
	@echo "Run Targets:"
	@echo "  make run-server    - Run server (port: $(SERVER_PORT))"
	@echo "  make run-cli       - Run CLI (connect to: $(CLI_HOST):$(CLI_PORT))"
	@echo "  make run-mcp-stdio - Run MCP STDIO server"
	@echo ""
	@echo "Development:"
	@echo "  make dev           - Hot reload mode (requires entr)"
	@echo "  make symlink       - Create symlinks in $(INSTALL_DIR)"
	@echo "  make unsymlink     - Remove symlinks"
	@echo "  make kill-ports    - Kill processes on common ports"
	@echo "  make update-ip     - Update IP addresses in config files"
	@echo ""
	@echo "Testing:"
	@echo "  make test          - Run all tests"
	@echo "  make test-coverage - Generate coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt           - Format Go code with goimports"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Run golangci-lint"
	@echo ""
	@echo "Installation:"
	@echo "  make install       - Install to $(INSTALL_DIR)"
	@echo "  make install-user  - Install to ~/.local/bin (no sudo)"
	@echo "  make uninstall     - Remove from $(INSTALL_DIR)"
	@echo "  INSTALL_DIR=/custom/path make install"
	@echo ""
	@echo "Release:"
	@echo "  make release       - Build for all platforms"
	@echo "  make deps          - Install Go dependencies"
	@echo "  make download-llama-deps - Download Llama libs"
	@echo ""
	@echo "Security:"
	@echo "  make generate-key  - Generate secure encryption keys"
	@echo ""
	@echo "Configuration:"
	@echo "  Current: v$(VERSION) on $(TARGET_OS)-$(TARGET_ARCH)"
	@echo "  Override: VERSION=2.0.0 SERVER_PORT=9000 make run"
