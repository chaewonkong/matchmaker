# API

### create ticket
```bash
curl -X POST http://localhost:8080/tickets \
  -H "Content-Type: application/json" \
  -d '{
    "ticket_id": "abc123",
    "player_ids": ["player1", "player2"],
    "time": "2025-07-23T15:04:05Z"
  }'
```