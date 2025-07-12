package queue_test

import (
	"testing"
	"time"

	"github.com/chaewonkong/matchmaker/schema"
	"github.com/chaewonkong/matchmaker/services/apiserver/queue"
	"github.com/stretchr/testify/assert"
)

func TestMatchingQueue(t *testing.T) {
	t.Run("Len 0", func(t *testing.T) {
		q := queue.New()
		assert.NotPanics(t, func() {
			assert.Equal(t, 0, q.Len(), "Expected queue length to be 0")
		})
	})

	t.Run("Enqueue", func(t *testing.T) {
		q := queue.New()
		assert.NotPanics(t, func() {
			q.Enqueue(schema.Ticket{})
			assert.Equal(t, 1, q.Len(), "Expected queue length to be 1 after enqueue")
		})
	})

	t.Run("Dequeue 1 item", func(t *testing.T) {
		q := queue.New()
		ticketID := "ticket1"
		assert.NotPanics(t, func() {
			q.Enqueue(schema.Ticket{ID: ticketID})
			assert.Equal(t, 1, q.Len(), "Expected queue length to be 1 before dequeue")
			ticket, ok := q.Dequeue()
			assert.True(t, ok, "Expected dequeue to succeed")
			assert.Equal(t, ticketID, ticket.ID, "Expected dequeued ticket ID to match")
			assert.Equal(t, 0, q.Len(), "Expected queue length to be 0 after dequeue")
		})
	})

	t.Run("enqueue, dequeue order check", func(t *testing.T) {
		q := queue.New()
		now := time.Now()
		tickets := []schema.Ticket{
			{ID: "1", Timestamp: now.Add(3 * time.Second)},
			{ID: "2", Timestamp: now.Add(2 * time.Second)},
			{ID: "3", Timestamp: now.Add(1 * time.Second)},
		}

		for _, tkt := range tickets {
			q.Enqueue(tkt)
		}

		assert.Equal(t, 3, q.Len(), "Expected queue length to be 3 after enqueueing 3 tickets")

		for i := 2; i >= 0; i-- {
			ticket, ok := q.Dequeue()
			assert.True(t, ok, "Expected dequeue to succeed")
			assert.Equal(t, tickets[i].ID, ticket.ID, "Expected dequeued ticket ID to match")
		}
	})
}
