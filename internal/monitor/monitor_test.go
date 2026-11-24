package monitor

import (
	"testing"
	"time"

	"jellyfin_sleep_inhibitor/internal/config"
	"jellyfin_sleep_inhibitor/internal/jellyfin"
	"jellyfin_sleep_inhibitor/internal/power"
)

func TestFilterActivePlayback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Power.IgnorePaused = false
	cfg.Power.PausedGracePeriodMinutes = 10

	m := NewMonitor(&cfg, nil, power.NewManager())

	sessions := []jellyfin.Session{
		{
			Id:       "1",
			UserName: "User1",
			NowPlayingItem: &jellyfin.NowPlayingItem{
				Name:      "Movie 1",
				MediaType: "Video",
			},
			PlayState: &jellyfin.PlayState{
				IsPaused:   false,
				PlayMethod: "DirectPlay",
			},
		},
		{
			Id:             "2",
			UserName:       "User2",
			NowPlayingItem: nil, // Browsing, not watching
			PlayState:      nil,
		},
		{
			Id:       "3",
			UserName: "User3",
			NowPlayingItem: &jellyfin.NowPlayingItem{
				Name:      "Movie 3",
				MediaType: "Video",
			},
			PlayState: &jellyfin.PlayState{
				IsPaused:   true,
				PlayMethod: "Transcode",
			},
		},
	}

	active := m.filterActivePlayback(sessions)
	// User1 is playing, User3 is paused (within 10m grace period)
	if len(active) != 2 {
		t.Fatalf("expected 2 active sessions, got %d", len(active))
	}

	if active[0].UserName != "User1" || active[0].IsPaused {
		t.Errorf("unexpected active[0]: %+v", active[0])
	}
	if active[1].UserName != "User3" || !active[1].IsPaused {
		t.Errorf("unexpected active[1]: %+v", active[1])
	}

	// Now test with IgnorePaused = true
	cfg.Power.IgnorePaused = true
	activeIgnorePaused := m.filterActivePlayback(sessions)
	if len(activeIgnorePaused) != 1 {
		t.Fatalf("expected 1 active session when ignoring paused, got %d", len(activeIgnorePaused))
	}
	if activeIgnorePaused[0].UserName != "User1" {
		t.Errorf("expected User1, got %s", activeIgnorePaused[0].UserName)
	}
}

func TestPausedGracePeriodExpiration(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Power.IgnorePaused = false
	cfg.Power.PausedGracePeriodMinutes = 5

	m := NewMonitor(&cfg, nil, power.NewManager())

	sessions := []jellyfin.Session{
		{
			Id:       "paused-session",
			UserName: "User1",
			NowPlayingItem: &jellyfin.NowPlayingItem{
				Name:      "Episode 1",
				MediaType: "Video",
			},
			PlayState: &jellyfin.PlayState{
				IsPaused:   true,
				PlayMethod: "DirectPlay",
			},
		},
	}

	// First check: should be active (starts timer)
	active1 := m.filterActivePlayback(sessions)
	if len(active1) != 1 {
		t.Fatalf("expected 1 active session on first check, got %d", len(active1))
	}

	// Simulate that it has been paused for 6 minutes (exceeding 5 min limit)
	m.pausedSince["paused-session"] = time.Now().Add(-6 * time.Minute)

	active2 := m.filterActivePlayback(sessions)
	if len(active2) != 0 {
		t.Fatalf("expected 0 active sessions after paused grace expired, got %d", len(active2))
	}
}
