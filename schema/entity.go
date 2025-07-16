package schema

import "time"

// Ticket represents a matchmaking ticket.
type Ticket struct {
	ID        string    `json:"id" validate:"required"`
	PlayerIDs []string  `json:"player_ids" validate:"required"`
	Timestamp time.Time `json:"timestamp" validate:"required"`
}

// Player represents a player.
type Player struct {
	ID string `json:"id"`
}

// Match represents a match
type Match struct {
	ID      string   `json:"id"`
	Tickets []Ticket `json:"tickets"`
}

// MatchResult represents the result of a match.
type MatchResult struct {
	MatchID string `json:"match_id"`
}
