package util

import (
	mcChat "neotunnel/util/mc-chat"
	"time"

	uuid "github.com/google/uuid"
)

type ServerStatus struct {
	Name              string
	Protocol          int
	PlayerCount       int
	MaxPlayerCount    int
	OnlinePlayerCount int
	SamplePlayers     []Player
	Description       mcChat.Message
	Favicon           string
	Delay             time.Duration
}

type Player struct {
	Name string    `json:"name"`
	ID   uuid.UUID `json:"id"`
}

type PingList struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int      `json:"max"`
		Online int      `json:"online"`
		Sample []Player `json:"sample"`
	} `json:"players"`
	Description mcChat.Message `json:"description"`
	FavIcon     string         `json:"favicon,omitempty"`
}

type HTTPResponse struct {
	Code int
	Data ServerStatus
}
