package schema

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type MatchingStrategy string

const (
	// player vs Environment
	PvE MatchingStrategy = "PvE"

	// Nop
	Nop MatchingStrategy = "Nop"

	// DualTeam red team vs blue team
	DualTeam MatchingStrategy = "DualTeam"
)

// QueueConfig represents the configuration for the matchmaking queue.
type QueueConfig struct {
	Name       string           `json:"name" yaml:"name"`
	ID         string           `json:"id" yaml:"id"`
	TeamLayout TeamLayout       `json:"team_layout" yaml:"team_layout"`
	Strategy   MatchingStrategy `json:"matching_strategy" yaml:"matching_strategy"`
}

// TeamLayout represents the layout of teams in the matchmaking system.
type TeamLayout struct {
	NumberOfTeams int `json:"number_of_teams" yaml:"number_of_teams"`
	TeamCapacity  int `json:"team_capacity" yaml:"team_capacity"`
}

// NewQueueConfig constructor
func NewQueueConfig() *QueueConfig {
	return &QueueConfig{}
}

// UnmarshalFromYAML reads the QueueConfig from a YAML file at the specified path.
func (c *QueueConfig) UnmarshalFromYAML(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open queue config file: %w", err)
	}

	err = yaml.NewDecoder(f).Decode(c)
	if err != nil {
		return fmt.Errorf("failed to decode queue config file: %w", err)
	}

	return nil
}
