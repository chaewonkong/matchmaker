package usecase

import (
	"fmt"

	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/queue"
)

// TicketService is responsible for creating and managing matchmaking tickets.
type TicketService struct {
	queue *queue.MatchingQueue
}

// NewTicketService creates a new TicketCreator instance.
func NewTicketService(q *queue.MatchingQueue) *TicketService {
	return &TicketService{
		queue: q,
	}
}

// Add creates a new matchmaking adds it to the queue.
func (t *TicketService) Add(ticket schema.Ticket) {
	t.queue.Enqueue(ticket)
}

// RemoveByID removes a matchmaking ticket from the queue by its ID.
func (t *TicketService) RemoveByID(ticketID string) error {
	_, ok := t.queue.RemoveTicketByID(ticketID)
	if !ok {
		return fmt.Errorf("ticket with ID %s not found", ticketID)
	}

	return nil
}
