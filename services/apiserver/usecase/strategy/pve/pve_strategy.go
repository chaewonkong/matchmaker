package pve

import (
	"fmt"

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
func (pe PvEStrategy) FindMatchCandidates() ([]schema.Match, error) {
	matchCandidates := []schema.Match{}
	cap := pe.QueueConfig.TeamLayout.TeamCapacity

	if cap < 1 {
		return nil, fmt.Errorf("error team capacity must be greater than 0")
	}

	for pe.Queue.Len() > 0 {

		candidate := []schema.Ticket{}
		rejected := []schema.Ticket{}
		slots := cap
		for {
			if slots == 0 {
				// full
				break
			}

			tkt, ok := pe.Queue.Dequeue()
			if !ok {
				break
			}

			n := len(tkt.PlayerIDs)
			if n > slots { // too many players
				rejected = append(rejected, tkt)
				continue
			}
			candidate = append(candidate, tkt)
			slots -= n
		}

		// add candidate
		if slots == 0 {
			matchID := uuid.NewString()
			teams := []schema.Team{
				{Index: 0, Tickets: candidate},
			}
			matchCandidates = append(matchCandidates, schema.Match{ID: matchID, Teams: teams})
		}

		// add rejected tickets to queue again
		for _, tkt := range rejected {
			pe.Queue.Enqueue(tkt)
		}
	}

	return matchCandidates, nil
}
