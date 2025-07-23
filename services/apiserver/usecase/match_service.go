package usecase

import (
	"fmt"

	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/apiserver/usecase/strategy"
	"github.com/chaewonkong/matchmaker/services/queue"
)

// MatchService match service
type MatchService struct {
	strategy strategy.Strategy
}

// NewMatchService constructor
func NewMatchService(cfg *schema.QueueConfig, q *queue.MatchingQueue) (*MatchService, error) {
	switch cfg.Strategy {
	case strategy.PvE:
		return &MatchService{
			strategy.PvEStrategy{
				Queue:       q,
				QueueConfig: cfg,
			},
		}, nil
	case strategy.Nop:
		return &MatchService{
			strategy.NopStrategy{},
		}, nil
	default:
		return nil, fmt.Errorf("error strategy mismatch")
	}
}

// FindAllMatchCandidates searches all possible match candidates
func (ms *MatchService) FindAllMatchCandidates() ([]schema.Match, error) {
	return ms.strategy.FindMatchCandidates()
}
