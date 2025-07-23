package strategy

import (
	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/queue"
	"github.com/google/uuid"
)

// PvEStrategy PvE strategy
type PvEStrategy struct {
	Queue       *queue.MatchingQueue
	QueueConfig *schema.QueueConfig
}

// FindMatchCandidates finds match candidates according to PvE strategy
func (pve PvEStrategy) FindMatchCandidates() ([]schema.Match, error) {
	matchCandidates := []schema.Match{}
	cap := pve.QueueConfig.TeamLayout.TeamCapacity
	n := pve.Queue.Len()

	for {
		if n < cap || cap < 1 {
			break
		}

		candidate := []schema.Ticket{}
		for range cap {
			t, ok := pve.Queue.Dequeue()
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
