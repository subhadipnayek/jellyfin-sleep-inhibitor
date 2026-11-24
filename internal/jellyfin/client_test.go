package jellyfin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSessions(t *testing.T) {
	mockResponse := `[
		{
			"Id": "session-1",
			"UserId": "user-1",
			"UserName": "Alice",
			"Client": "Jellyfin Web",
			"DeviceName": "Chrome Windows",
			"IsActive": true,
			"NowPlayingItem": {
				"Id": "item-1",
				"Name": "Interstellar",
				"Type": "Movie",
				"MediaType": "Video"
			},
			"PlayState": {
				"PositionTicks": 10000000,
				"IsPaused": false,
				"PlayMethod": "DirectPlay"
			}
		},
		{
			"Id": "session-2",
			"UserId": "user-2",
			"UserName": "Bob",
			"Client": "Jellyfin Android",
			"DeviceName": "Pixel 8",
			"IsActive": true,
			"NowPlayingItem": null,
			"PlayState": null
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Sessions" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-Emby-Token") != "secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token", 5)
	sessions, err := client.FetchSessions(context.Background())
	if err != nil {
		t.Fatalf("FetchSessions failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	s1 := sessions[0]
	if s1.UserName != "Alice" || s1.NowPlayingItem == nil || s1.NowPlayingItem.Name != "Interstellar" {
		t.Errorf("session 1 mismatch: %+v", s1)
	}
	if s1.PlayState.IsPaused {
		t.Errorf("expected session 1 not paused")
	}

	s2 := sessions[1]
	if s2.UserName != "Bob" || s2.NowPlayingItem != nil {
		t.Errorf("session 2 mismatch: %+v", s2)
	}
}

func TestFetchSessionsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "wrong-token", 5)
	_, err := client.FetchSessions(context.Background())
	if err == nil {
		t.Errorf("expected error on unauthorized, got nil")
	}
}
