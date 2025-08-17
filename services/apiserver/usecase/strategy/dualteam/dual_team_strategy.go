package dualteam

import (
	"container/list"

	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/apiserver/usecase/strategy"
	"github.com/chaewonkong/matchmaker/services/queue"
)

var _ strategy.Strategy = (*DualteamStrategy)(nil)

// DualteamStrategy dual team layout strategy
type DualteamStrategy struct {
	Queue       *queue.MatchingQueue
	QueueConfig *schema.QueueConfig
}

// FindMatchCandidates finds match candidates in dual team layout
func (d *DualteamStrategy) FindMatchCandidates() ([]schema.Match, error) {
	candidates := []schema.Match{}
	// teams := []schema.Team{}
	numTeams := d.QueueConfig.TeamLayout.NumberOfTeams
	teamCap := d.QueueConfig.TeamLayout.TeamCapacity

	// generate teams first
	teams := list.New()

	for d.Queue.Len() > 0 {
		team := schema.Team{}

		for len(team.Tickets) < teamCap {
			tkt, ok := d.Queue.Dequeue()
			if !ok {
				break
			}

			team.Tickets = append(team.Tickets, tkt)
		}
		teams.PushBack(team)
	}

	for teams.Len() > 0 {
		m := schema.Match{}

		// TODO: shuffle teams, or make each team in match candidate fair and even.
		// FIXME: this code discards team when match team layout is not satisfied.
		for i := range numTeams {
			if teams.Len() > 0 {
				team := teams.Front().Value.(schema.Team)
				team.Index = i
				m.Teams = append(m.Teams, team)
			}
		}

		// if m has enough teams, append m to candidates
		if len(m.Teams) == numTeams {
			candidates = append(candidates, m)
		}
		// if len(m.Teams) < numTeams, discard m
	}

	return candidates, nil
}
