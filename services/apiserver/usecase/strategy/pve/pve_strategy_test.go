package pve_test

import (
	"testing"
	"time"

	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/apiserver/usecase/strategy"
	"github.com/chaewonkong/matchmaker/services/apiserver/usecase/strategy/pve"
	"github.com/chaewonkong/matchmaker/services/queue"
	"github.com/stretchr/testify/assert"
)

func TestPvEStrategyFindMatchCandidates(t *testing.T) {
	t.Run("Finds match candidates successfully", func(t *testing.T) {
		// given
		queue := queue.New()
		cfg := &schema.QueueConfig{
			ID:   "queue1",
			Name: "player vs environment",
			TeamLayout: schema.TeamLayout{
				NumberOfTeams: 1,
				TeamCapacity:  4,
			},
			Strategy: strategy.PvE,
		}
		s := &pve.PvEStrategy{
			Queue:       queue,
			QueueConfig: cfg,
		}
		now := time.Now()

		t1 := schema.Ticket{
			ID:        "1",
			PlayerIDs: []string{"p1", "p2"},
			Timestamp: now,
		}
		t2 := schema.Ticket{
			ID:        "2",
			PlayerIDs: []string{"p3", "p4", "p5"},
			Timestamp: now.Add(1 * time.Second),
		}
		t3 := schema.Ticket{
			ID:        "3",
			PlayerIDs: []string{"p6"},
			Timestamp: now.Add(2 * time.Second),
		}
		t4 := schema.Ticket{
			ID:        "4",
			PlayerIDs: []string{"p7"},
			Timestamp: now.Add(3 * time.Second),
		}
		t5 := schema.Ticket{
			ID:        "5",
			PlayerIDs: []string{"p8"},
			Timestamp: now.Add(4 * time.Second),
		}

		// when
		queue.Enqueue(t1)
		queue.Enqueue(t2)
		queue.Enqueue(t3)
		queue.Enqueue(t4)
		queue.Enqueue(t5)

		assert.Equal(t, 5, queue.Len())

		// then
		results, err := s.FindMatchCandidates()
		assert.NoError(t, err)

		for _, match := range results {
			cnt := 0

			for _, team := range match.Teams {
				for _, tkt := range team.Tickets {
					cnt += len(tkt.PlayerIDs)
				}
			}
			assert.Equal(t, 4, cnt)
		}

	})
}
