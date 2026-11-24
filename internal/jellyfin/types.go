package jellyfin

import "time"

// PlayState represents the playback state of a session
type PlayState struct {
	PositionTicks    int64  `json:"PositionTicks"`
	CanSeek          bool   `json:"CanSeek"`
	IsPaused         bool   `json:"IsPaused"`
	IsMuted          bool   `json:"IsMuted"`
	VolumeLevel      int    `json:"VolumeLevel"`
	PlayMethod       string `json:"PlayMethod"` // DirectPlay, DirectStream, Transcode
	RepeatMode       string `json:"RepeatMode"`
	PlaybackRate     float64 `json:"PlaybackRate"`
}

// NowPlayingItem represents the media currently being played
type NowPlayingItem struct {
	Name            string   `json:"Name"`
	Id              string   `json:"Id"`
	Type            string   `json:"Type"` // Movie, Episode, Audio, etc.
	MediaType       string   `json:"MediaType"` // Video, Audio, Book
	RunTimeTicks    int64    `json:"RunTimeTicks"`
	SeriesName      string   `json:"SeriesName,omitempty"`
	SeasonName      string   `json:"SeasonName,omitempty"`
	IndexNumber     int      `json:"IndexNumber,omitempty"` // Episode number
	ParentIndexNum  int      `json:"ParentIndexNumber,omitempty"` // Season number
	Artists         []string `json:"Artists,omitempty"`
}

// Session represents an active Jellyfin client session
type Session struct {
	Id               string          `json:"Id"`
	UserId           string          `json:"UserId"`
	UserName         string          `json:"UserName"`
	Client           string          `json:"Client"`
	LastActivityDate time.Time       `json:"LastActivityDate"`
	DeviceName       string          `json:"DeviceName"`
	DeviceId         string          `json:"DeviceId"`
	ApplicationVersion string        `json:"ApplicationVersion"`
	IsActive         bool            `json:"IsActive"`
	NowPlayingItem   *NowPlayingItem `json:"NowPlayingItem"`
	PlayState        *PlayState      `json:"PlayState"`
}

// ActivePlaybackSummary provides details of who is watching what
type ActivePlaybackSummary struct {
	UserName   string
	DeviceName string
	Client     string
	MediaTitle string
	MediaType  string
	PlayMethod string
	IsPaused   bool
}
