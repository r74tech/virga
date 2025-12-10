package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/r74tech/virga/internal/server"
	"github.com/r74tech/virga/internal/server/config"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/listener"
	"github.com/r74tech/virga/internal/shared/logger"
)

// Global variable for debug mode
var debugMode = false

func main() {
	// Set command-line flags
	configFile := flag.String("config", "configs/server.yaml", "Path to configuration file")
	debugFlag := flag.Bool("debug", false, "Enable debug mode with verbose logging")
	logLevel := flag.String("log-level", "", "Set log level (debug, info, warn, error, off)")
	noLog := flag.Bool("no-log", false, "Disable all logging")
	flag.Parse()

	// Set log level from environment or flag
	logger.SetGlobalLogLevelFromEnv("VIRGA_SERVER_LOG_LEVEL")

	// Override with command line flag
	if *noLog {
		logger.SetGlobalLogLevel(logger.OFF)
	} else if *logLevel != "" {
		logger.SetGlobalLogLevelFromString(*logLevel)
	}

	// Set debug mode
	debugMode = *debugFlag
	if debugMode {
		logger.SetGlobalLogLevel(logger.DEBUG)
		logger.Info("Debug mode enabled - verbose logging activated")
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override log level with debug flag or config
	if debugMode {
		cfg.Server.LogLevel = "debug"
	} else if *logLevel != "" {
		cfg.Server.LogLevel = *logLevel
	}

	// Apply log level from config if not already set
	if cfg.Server.LogLevel != "" && *logLevel == "" && !*debugFlag && !*noLog {
		logger.SetGlobalLogLevelFromString(cfg.Server.LogLevel)
	}

	logger.Info("Server log level set to %s", logger.GetGlobalLogger().GetLevel().String())

	// Initialize database
	db, err := database.Initialize(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create C2 server instance
	c2Server := server.NewServer(cfg, db)
	// Set debug mode
	c2Server.SetDebugMode(debugMode)

	// Set listeners
	for _, listenerConfig := range cfg.Listeners {
		if debugMode {
			logger.Debug("Creating listener: %s (%s://%s:%d)",
				listenerConfig.Name,
				listenerConfig.Type,
				listenerConfig.BindAddress,
				listenerConfig.Port)
		}

		listener, err := listener.NewListener(listenerConfig)
		if err != nil {
			logger.Error("Failed to create listener %s: %v", listenerConfig.Name, err)
			continue
		}

		// Pass debug information to listener
		if l, ok := listener.(interface{ SetDebugMode(bool) }); ok && debugMode {
			l.SetDebugMode(debugMode)
		}

		err = c2Server.AddListener(listener)
		if err != nil {
			logger.Error("Failed to add listener %s: %v", listenerConfig.Name, err)
		}
	}

	// Start server
	go c2Server.Start()
	logger.Info("Virga C2 Server started. Press Ctrl+C to exit.")

	// Signal handling (Ctrl+C, etc.)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// Shutdown processing
	logger.Info("\nShutting down...")
	c2Server.Shutdown()
}
