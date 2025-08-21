package strategy

import "github.com/chaewonkong/matchmaker/schema"

// Strategy match candidate finder algorithm
type Strategy interface {
	// FindMatchCandidates finds match candidates
	FindMatchCandidates() ([]schema.Match, error)
}

// NopStrategy nop
type NopStrategy struct{}

// FindMatchCandidates nop
func (NopStrategy) FindMatchCandidates() ([]schema.Match, error) {
	return nil, nil
}
