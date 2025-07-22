package usecase

import (
	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/queue"
)

// MatchService match service
type MatchService struct {
	queue       *queue.MatchingQueue
	queueConfig *schema.QueueConfig
}

// NewMatchService constructor
func NewMatchService(cfg *schema.QueueConfig, q *queue.MatchingQueue) *MatchService {
	return &MatchService{queue: q, queueConfig: cfg}
}

// FindAllMatchCandidates searches all possible match candidates
func (ms *MatchService) FindAllMatchCandidates() (schema.Match, error) {
	// Retrieve all match candidates from the queue

	ticket, ok := ms.queue.Dequeue()
	if ok {
		return schema.Match{
			ID:      "111",
			Tickets: []schema.Ticket{ticket},
		}, nil
	}

	// TODO: use config
	return schema.Match{}, nil
}
