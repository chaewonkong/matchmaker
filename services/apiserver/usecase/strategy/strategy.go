package strategy

import "github.com/chaewonkong/matchmaker/schema"

// Strategy match candidate finder algorithm
type Strategy interface {
	// FindMatchCandidates finds match candidates
	FindMatchCandidates() ([]schema.Match, error)
}

const (
	// player vs Environment
	PvE schema.MatchingStrategy = "PvE"

	// Nop
	Nop schema.MatchingStrategy = "Nop"
)

// NopStrategy nop
type NopStrategy struct{}

// FindMatchCandidates nop
func (NopStrategy) FindMatchCandidates() ([]schema.Match, error) {
	return nil, nil
}
