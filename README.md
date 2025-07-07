# 🎮 matchmaker

A lightweight and high-performance matchmaker service written in Go.  
Designed for real-time game matchmaking with concurrent player support.

# Components
## API Server
The API server provides a RESTful interface for managing matchmaking tickets, match candidates, player acknowledgements, and game results.

### REST API Endpoints

Below is a summary of the REST API endpoints and their purposes:

| Description                                | HTTP Method | Endpoint                                |
|--------------------------------------------|-------------|-----------------------------------------|
| Create matchmaking ticket                  | POST        | `/tickets`                              |
| Cancel matchmaking ticket                  | DELETE      | `/tickets/{ticket_id}`                  |
| List current match candidates              | GET         | `/matches/candidates`                   |
| Create or update player acknowledgement    | PUT         | `/matches/{match_id}/acknowledgement`   |
| Submit game result (win/loss)              | POST        | `/match-results`                        |
