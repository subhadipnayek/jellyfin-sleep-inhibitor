package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows/svc"
	"jellyfin_sleep_inhibitor/internal/config"
	"jellyfin_sleep_inhibitor/internal/jellyfin"
	"jellyfin_sleep_inhibitor/internal/monitor"
	"jellyfin_sleep_inhibitor/internal/power"
	"jellyfin_sleep_inhibitor/internal/service"
)

const Version = "1.0.0"

func main() {
	isWinService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to determine service execution context: %v", err)
	}

	if isWinService {
		runAsWindowsService()
		return
	}

	// Interactive CLI execution
	runCLI()
}

func runAsWindowsService() {
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Service configuration load failed: %v", err)
	}

	setupLogging(cfg, true)
	if err := service.RunService(cfg); err != nil {
		log.Fatalf("Service terminated with error: %v", err)
	}
}

func runCLI() {
	configFlag := flag.String("config", "", "Path to config.json")
	flag.Usage = printHelp
	flag.Parse()

	args := flag.Args()
	cmd := "run"
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "install":
		if err := service.Install(); err != nil {
			fmt.Printf("[ERROR] Install failed: %v\n", err)
			os.Exit(1)
		}
	case "uninstall":
		if err := service.Uninstall(); err != nil {
			fmt.Printf("[ERROR] Uninstall failed: %v\n", err)
			os.Exit(1)
		}
	case "start":
		if err := service.Start(); err != nil {
			fmt.Printf("[ERROR] Start failed: %v\n", err)
			os.Exit(1)
		}
	case "stop":
		if err := service.Stop(); err != nil {
			fmt.Printf("[ERROR] Stop failed: %v\n", err)
			os.Exit(1)
		}
	case "status":
		status, err := service.Status()
		if err != nil {
			fmt.Printf("[ERROR] Failed to query service status: %v\n", err)
		} else {
			fmt.Printf("Service Status: %s\n", status)
		}

		// Also check Jellyfin connectivity if config is available
		cfg, err := config.LoadConfig(*configFlag)
		if err != nil {
			fmt.Printf("Configuration: %v\n", err)
		} else {
			fmt.Printf("Jellyfin Server: %s\n", cfg.Jellyfin.ServerURL)
			client := jellyfin.NewClient(cfg.Jellyfin.ServerURL, cfg.Jellyfin.APIKey, 5)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			msg, err := client.CheckConnectivity(ctx)
			if err != nil {
				fmt.Printf("Jellyfin Test: [FAILED] %v\n", err)
			} else {
				fmt.Printf("Jellyfin Test: [OK] %s\n", msg)
			}
		}
	case "run":
		cfg, err := config.LoadConfig(*configFlag)
		if err != nil {
			fmt.Printf("[ERROR] %v\n", err)
			fmt.Println("\nTip: Copy config.json.example to config.json and set your Jellyfin API key.")
			os.Exit(1)
		}

		setupLogging(cfg, false)
		runInteractive(cfg)
	case "version":
		fmt.Printf("Jellyfin Sleep Inhibitor v%s (Windows x64)\n", Version)
	case "help":
		printHelp()
	default:
		fmt.Printf("Unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func runInteractive(cfg *config.Config) {
	fmt.Println("=====================================================")
	fmt.Printf(" Jellyfin Sleep Inhibitor v%s (Console Mode)\n", Version)
	fmt.Println(" Press Ctrl+C to stop")
	fmt.Println("=====================================================")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	client := jellyfin.NewClient(
		cfg.Jellyfin.ServerURL,
		cfg.Jellyfin.APIKey,
		cfg.Jellyfin.RequestTimeoutSeconds,
	)
	pwr := power.NewManager()
	mon := monitor.NewMonitor(cfg, client, pwr)

	go mon.Run(ctx)

	<-sigChan
	fmt.Println("\nReceived shutdown signal, releasing power state...")
	cancel()
	_ = pwr.AllowSleep()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Shutdown complete. Goodbye.")
}

func setupLogging(cfg *config.Config, isService bool) {
	log.SetFlags(log.Ldate | log.Ltime)

	if !cfg.Logging.LogToFile && !isService {
		return
	}

	logPath := cfg.Logging.LogFilePath
	if !filepath.IsAbs(logPath) {
		logPath = filepath.Join(config.GetExeDir(), logPath)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		if !isService {
			fmt.Printf("[WARN] Failed to open log file (%s): %v\n", logPath, err)
		}
		return
	}

	if isService {
		log.SetOutput(logFile)
	} else {
		// In console mode, write to both terminal and file
		multi := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multi)
	}
}

func printHelp() {
	fmt.Printf(`Jellyfin Sleep Inhibitor v%s

Keeps Windows PC awake while Jellyfin media is being actively streamed,
and allows sleep when all users stop watching.

Usage:
  jellyfin-sleep-inhibitor [options] <command>

Commands:
  run          Run interactively in console (default)
  install      Install as a Windows Service (requires Administrator)
  uninstall    Uninstall the Windows Service (requires Administrator)
  start        Start the Windows Service
  stop         Stop the Windows Service
  status       Check service status and test Jellyfin connection
  version      Print version information
  help         Show this help screen

Options:
  -config      Path to custom config.json (default: config.json in app directory)

Examples:
  # Test interactively in console:
  jellyfin-sleep-inhibitor run

  # Install and start as a background Windows Service:
  jellyfin-sleep-inhibitor install
  jellyfin-sleep-inhibitor start
`, Version)
}
