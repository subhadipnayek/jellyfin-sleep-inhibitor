package jellyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	serverURL  string
	apiKey     string
	httpClient *http.Client
}

func NewClient(serverURL, apiKey string, timeoutSeconds int) *Client {
	url := strings.TrimRight(serverURL, "/")
	return &Client{
		serverURL: url,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// FetchSessions retrieves all current sessions from Jellyfin
func (c *Client) FetchSessions(ctx context.Context) ([]Session, error) {
	endpoint := fmt.Sprintf("%s/Sessions", c.serverURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create sessions request: %w", err)
	}

	req.Header.Set("X-Emby-Token", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "JellyfinSleepInhibitor/1.0.0 (Windows Service)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jellyfin connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("authentication failed (HTTP %d): verify your API key in config.json", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jellyfin server returned status HTTP %d", resp.StatusCode)
	}

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions response: %w", err)
	}

	return sessions, nil
}

// CheckConnectivity verifies server URL and API key
func (c *Client) CheckConnectivity(ctx context.Context) (string, error) {
	endpoint := fmt.Sprintf("%s/System/Info/Public", c.serverURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot connect to %s: %w", c.serverURL, err)
	}
	resp.Body.Close()

	// Now check /Sessions with auth
	_, err = c.FetchSessions(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully connected to Jellyfin at %s", c.serverURL), nil
}
