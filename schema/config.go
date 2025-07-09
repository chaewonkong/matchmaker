package schema

// QueueConfig represents the configuration for the matchmaking queue.
type QueueConfig struct {
	Name       string     `json:"name"`
	ID         string     `json:"id"`
	TeamLayout TeamLayout `json:"team_layout"`
}

// TeamLayout represents the layout of teams in the matchmaking system.
type TeamLayout struct {
	NumberOfTeams int `json:"number_of_teams"`
	TeamCapacity  int `json:"team_capacity"`
}
