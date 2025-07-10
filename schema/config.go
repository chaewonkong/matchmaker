package schema

// QueueConfig represents the configuration for the matchmaking queue.
type QueueConfig struct {
	Name       string     `json:"name" yaml:"name"`
	ID         string     `json:"id" yaml:"id"`
	TeamLayout TeamLayout `json:"team_layout" yaml:"team_layout"`
}

// TeamLayout represents the layout of teams in the matchmaking system.
type TeamLayout struct {
	NumberOfTeams int `json:"number_of_teams" yaml:"number_of_teams"`
	TeamCapacity  int `json:"team_capacity" yaml:"team_capacity"`
}
