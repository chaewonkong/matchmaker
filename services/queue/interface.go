package queue

import (
	"context"

	"github.com/chaewonkong/matchmaker/schema"
)

// Queue where tickets wait to be matched
type Queue interface {
	// Add add requested ticket to the queue
	Add(ctx context.Context, ticket schema.Ticket) error

	// Fetch fetches top n number of tickets from queue.
	Fetch(ctx, n int) ([]schema.Ticket, error)
}
