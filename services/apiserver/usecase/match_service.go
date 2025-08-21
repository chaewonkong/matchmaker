package usecase

import (
	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/apiserver/usecase/strategy"
	"github.com/chaewonkong/matchmaker/services/apiserver/usecase/strategy/dualteam"
	"github.com/chaewonkong/matchmaker/services/apiserver/usecase/strategy/pve"
	"github.com/chaewonkong/matchmaker/services/queue"
)

// MatchService match service
type MatchService struct {
	strategy strategy.Strategy
}

// NewMatchService constructor
func NewMatchService(cfg *schema.QueueConfig, q *queue.MatchingQueue) (*MatchService, error) {
	switch cfg.Strategy {
	case schema.PvE:
		return &MatchService{
			pve.PvEStrategy{
				Queue:       q,
				QueueConfig: cfg,
			},
		}, nil
	case schema.DualTeam:
		return &MatchService{
			dualteam.DualteamStrategy{
				Queue:       q,
				QueueConfig: cfg,
			},
		}, nil
	default: // nop
		return &MatchService{
			strategy.NopStrategy{},
		}, nil
	}
}

// FindAllMatchCandidates searches all possible match candidates
func (ms *MatchService) FindAllMatchCandidates() ([]schema.Match, error) {
	return ms.strategy.FindMatchCandidates()
}
