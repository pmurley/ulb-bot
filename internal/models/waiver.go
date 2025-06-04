package models

import (
	"time"
)

// Waiver represents a player on waivers
type Waiver struct {
	PlayerName string    // Name of the player on waivers
	TeamName   string    // Team that owns the player
	UserID     string    // Discord user ID who initiated the DFA
	StartTime  time.Time // When the waiver period started
	EndTime    time.Time // When the waiver period ends (7 days later)
	MessageID  string    // Discord message ID to reply to
	ChannelID  string    // Discord channel ID where command was issued
	Processed  bool      // Whether this waiver has been processed
}

// IsExpired checks if the waiver period has expired
func (w *Waiver) IsExpired() bool {
	return time.Now().After(w.EndTime)
}
