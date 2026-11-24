//go:build windows

package service

import (
	"context"
	"log"
	"time"

	"golang.org/x/sys/windows/svc"
	"jellyfin_sleep_inhibitor/internal/config"
	"jellyfin_sleep_inhibitor/internal/jellyfin"
	"jellyfin_sleep_inhibitor/internal/monitor"
	"jellyfin_sleep_inhibitor/internal/power"
)

const ServiceName = "JellyfinSleepInhibitor"
const ServiceDisplayName = "Jellyfin Sleep Inhibitor Service"
const ServiceDescription = "Monitors Jellyfin media playback and prevents Windows system sleep while users are streaming."

type WindowsService struct {
	cfg *config.Config
}

func NewWindowsService(cfg *config.Config) *WindowsService {
	return &WindowsService{cfg: cfg}
}

func (ws *WindowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := jellyfin.NewClient(
		ws.cfg.Jellyfin.ServerURL,
		ws.cfg.Jellyfin.APIKey,
		ws.cfg.Jellyfin.RequestTimeoutSeconds,
	)
	pwr := power.NewManager()
	mon := monitor.NewMonitor(ws.cfg, client, pwr)

	// Start monitor in background goroutine
	go mon.Run(ctx)

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	log.Println("[INFO] Jellyfin Sleep Inhibitor Service started successfully.")

	for {
		select {
		case req := <-r:
			switch req.Cmd {
			case svc.Interrogate:
				changes <- req.CurrentStatus
				// Testing reception of request
				time.Sleep(100 * time.Millisecond)
				changes <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Println("[INFO] Service stop/shutdown request received.")
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				// Allow small time for monitor cleanup
				time.Sleep(500 * time.Millisecond)
				changes <- svc.Status{State: svc.Stopped}
				return
			default:
				log.Printf("[WARN] Unexpected service control request #%d", req)
			}
		}
	}
}

func RunService(cfg *config.Config) error {
	return svc.Run(ServiceName, NewWindowsService(cfg))
}
