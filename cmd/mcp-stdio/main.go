package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/config"
	"github.com/r74tech/virga/internal/server/database"
	mcpconfig "github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/mcp/stdio"
	"github.com/r74tech/virga/internal/server/session"
)

func main() {
	// Set command-line flags
	configFile := flag.String("config", "server.yaml", "Path to configuration file")
	debugFlag := flag.Bool("debug", false, "Enable debug mode with verbose logging")
	flag.Parse()

	// Set debug mode
	if *debugFlag {
		log.SetOutput(os.Stderr)
		log.Println("MCP STDIO Server - Debug mode enabled")
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.Initialize(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize manager
	sessionManager := session.NewManager()
	beaconManager := beacons.NewManager()

	// MCP server configuration
	mcpConfig := &mcpconfig.Config{
		Name:         "Virga MCP Server",
		Version:      "1.0.0",
		StdioEnabled: true,
	}

	// Initialize MCP server (create STDIO server directly)
	mcpServer, err := stdio.NewServer(mcpConfig, beaconManager, sessionManager, db)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Start MCP server in STDIO mode
	log.Println("Starting MCP server in STDIO mode...")
	ctx := context.Background()
	if err := mcpServer.Start(ctx); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
