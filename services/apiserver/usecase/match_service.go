package usecase

import (
	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/queue"
	"github.com/google/uuid"
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
func (ms *MatchService) FindAllMatchCandidates() ([]schema.Match, error) {
	// Retrieve all match candidates from the queue

	matchCandidates := []schema.Match{}
	cap := ms.queueConfig.TeamLayout.TeamCapacity
	n := ms.queue.Len()

	for {
		if n < cap || cap < 1 {
			break
		}

		candidate := []schema.Ticket{}
		for range cap {
			t, ok := ms.queue.Dequeue()
			if !ok {
				// TODO: log.warn
				break
			}
			n-- // decrement n
			candidate = append(candidate, t)
		}

		// add candidate
		matchID := uuid.New().String()
		matchCandidates = append(matchCandidates, schema.Match{ID: matchID, Tickets: candidate})
	}

	return matchCandidates, nil
}
