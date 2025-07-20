package schema

import "time"

// TicketRequest create ticket request
type TicketRequest struct {
	ID        string   `json:"ticket_id" validate:"required"`
	PlayerIDs []string `json:"player_ids" validate:"required"`
	Time      string   `json:"time" validate:"required"` // ISO 8601 format
}

// ToTicket converts TicketRequest to Ticket
func (t *TicketRequest) ToTicket() (tkt Ticket, err error) {
	// parse ISO 8601 time format
	timestamp, err := time.Parse(time.RFC3339, t.Time)
	if err != nil {
		return
	}

	tkt.ID = t.ID
	tkt.PlayerIDs = t.PlayerIDs
	tkt.Timestamp = timestamp

	return
}
