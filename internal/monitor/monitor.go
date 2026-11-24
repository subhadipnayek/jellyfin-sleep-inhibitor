package monitor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"jellyfin_sleep_inhibitor/internal/config"
	"jellyfin_sleep_inhibitor/internal/jellyfin"
	"jellyfin_sleep_inhibitor/internal/power"
)

type Monitor struct {
	cfg          *config.Config
	client       *jellyfin.Client
	powerManager *power.Manager
	lastActive   time.Time
	pausedSince  map[string]time.Time
	consecErrors int
}

func NewMonitor(cfg *config.Config, client *jellyfin.Client, powerManager *power.Manager) *Monitor {
	return &Monitor{
		cfg:          cfg,
		client:       client,
		powerManager: powerManager,
		lastActive:   time.Now(),
		pausedSince:  make(map[string]time.Time),
	}
}

// Run executes the monitoring loop until ctx is canceled
func (m *Monitor) Run(ctx context.Context) {
	log.Printf("[INFO] Starting Jellyfin Sleep Monitor (Interval: %ds, Idle Grace: %dm)",
		m.cfg.Jellyfin.PollingIntervalSeconds,
		m.cfg.Power.IdleGracePeriodMinutes)

	ticker := time.NewTicker(time.Duration(m.cfg.Jellyfin.PollingIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Initial check immediately on startup
	m.checkOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] Shutting down monitor. Releasing power locks...")
			_ = m.powerManager.AllowSleep()
			return
		case <-ticker.C:
			m.checkOnce(ctx)
		}
	}
}

func (m *Monitor) checkOnce(ctx context.Context) {
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(m.cfg.Jellyfin.RequestTimeoutSeconds)*time.Second)
	defer cancel()

	sessions, err := m.client.FetchSessions(reqCtx)
	if err != nil {
		m.consecErrors++
		log.Printf("[WARN] Error communicating with Jellyfin (#%d): %v", m.consecErrors, err)

		if m.cfg.Power.KeepAwakeOnNetworkError {
			if !m.powerManager.IsInhibited() {
				log.Println("[INFO] Fail-safe active: Inhibiting sleep due to network error.")
				_ = m.powerManager.InhibitSleep(m.cfg.Power.RequireAwayMode)
			}
		}
		return
	}

	if m.consecErrors > 0 {
		log.Printf("[INFO] Jellyfin connection restored after %d failed attempts.", m.consecErrors)
		m.consecErrors = 0
	}

	activeList := m.filterActivePlayback(sessions)

	if len(activeList) > 0 {
		m.lastActive = time.Now()
		var descriptions []string
		for _, s := range activeList {
			status := "Playing"
			if s.IsPaused {
				status = "Paused"
			}
			descriptions = append(descriptions, fmt.Sprintf("%s (%s on %s - %s [%s])",
				s.UserName, s.MediaTitle, s.DeviceName, status, s.PlayMethod))
		}

		if !m.powerManager.IsInhibited() {
			log.Printf("[ACTIVE] %d active playback session(s) detected: %s. Inhibiting Windows sleep.",
				len(activeList), strings.Join(descriptions, "; "))
			if err := m.powerManager.InhibitSleep(m.cfg.Power.RequireAwayMode); err != nil {
				log.Printf("[ERROR] Failed to inhibit sleep: %v", err)
			}
		} else {
			// Periodic status log
			log.Printf("[AWAKE] Streaming active: %s", strings.Join(descriptions, "; "))
		}
	} else {
		// No active playback
		gracePeriod := time.Duration(m.cfg.Power.IdleGracePeriodMinutes) * time.Minute

		if m.powerManager.IsInhibited() {
			timeSinceActive := time.Since(m.lastActive)
			if timeSinceActive < gracePeriod {
				remaining := gracePeriod - timeSinceActive
				log.Printf("[IDLE GRACE] No active playback. Keeping PC awake during grace period (%s remaining)...",
					remaining.Round(time.Second))
			} else {
				log.Println("[SLEEP ALLOWED] Idle grace period expired and all sessions are idle. Releasing sleep inhibitor.")
				if err := m.powerManager.AllowSleep(); err != nil {
					log.Printf("[ERROR] Failed to release sleep inhibitor: %v", err)
				}
			}
		}
	}
}

func (m *Monitor) filterActivePlayback(sessions []jellyfin.Session) []jellyfin.ActivePlaybackSummary {
	var active []jellyfin.ActivePlaybackSummary
	currentSessionIDs := make(map[string]bool)

	for _, s := range sessions {
		if s.NowPlayingItem == nil || s.PlayState == nil {
			continue
		}

		currentSessionIDs[s.Id] = true
		title := s.NowPlayingItem.Name
		if s.NowPlayingItem.SeriesName != "" {
			title = fmt.Sprintf("%s - %s", s.NowPlayingItem.SeriesName, s.NowPlayingItem.Name)
		}

		user := s.UserName
		if user == "" {
			user = "Guest"
		}
		device := s.DeviceName
		if device == "" {
			device = s.Client
		}

		if s.PlayState.IsPaused {
			if m.cfg.Power.IgnorePaused {
				continue
			}

			// Track paused time
			pauseStart, exists := m.pausedSince[s.Id]
			if !exists {
				pauseStart = time.Now()
				m.pausedSince[s.Id] = pauseStart
			}

			maxPausedDuration := time.Duration(m.cfg.Power.PausedGracePeriodMinutes) * time.Minute
			if time.Since(pauseStart) > maxPausedDuration {
				// Paused too long, do not count as active
				continue
			}
		} else {
			// Not paused, remove from paused tracking
			delete(m.pausedSince, s.Id)
		}

		active = append(active, jellyfin.ActivePlaybackSummary{
			UserName:   user,
			DeviceName: device,
			Client:     s.Client,
			MediaTitle: title,
			MediaType:  s.NowPlayingItem.MediaType,
			PlayMethod: s.PlayState.PlayMethod,
			IsPaused:   s.PlayState.IsPaused,
		})
	}

	// Clean up stale sessions from paused tracking
	for id := range m.pausedSince {
		if !currentSessionIDs[id] {
			delete(m.pausedSince, id)
		}
	}

	return active
}
