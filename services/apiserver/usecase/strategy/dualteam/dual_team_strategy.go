package dualteam

import (
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

type team struct {
	tickets []schema.Ticket
	size    int
}

// FindMatchCandidates finds match candidates in dual team layout
func (d *DualteamStrategy) FindMatchCandidates() ([]schema.Match, error) {
	return nil, nil
}

// ?:팀전인 경우에도 match-candidate를 동일한 return type으로 반환할 것인가?
