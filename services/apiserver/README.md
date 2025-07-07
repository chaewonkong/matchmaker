## REST API Handlers

Below is a summary of the REST API endpoints with their purpose and recommended Go handler function names:

| Description                        | HTTP Method | Endpoint                                | Handler Function Name               |
|----------------------------------|-------------|---------------------------------------|-----------------------------------|
| Create matchmaking ticket         | POST        | `/tickets`                            | `CreateTicket`                    |
| Cancel matchmaking ticket         | DELETE      | `/tickets/{ticket_id}`                | `DeleteTicketByID`                |
| List current match candidates     | GET         | `/matches/candidates`                 | `FindAllMatchCandidates`          |
| Create or update player acknowledgement (idempotent) | PUT         | `/matches/{match_id}/ack` | `CreateOrUpdateMatchAck` |
| Submit game result (win/loss)     | POST        | `/match-results`                      | `CreateMatchResult`               |
