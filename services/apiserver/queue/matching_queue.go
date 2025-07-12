package queue

import "github.com/chaewonkong/matchmaker/schema"

// MatchingQueue is a priority queue for matchmaking tickets.
type MatchingQueue struct {
	queue queue
}

// New creates a new MatchingQueue instance.
func New() *MatchingQueue {
	return &MatchingQueue{
		queue: queue{},
	}
}

// Len returns the number of tickets in the queue.
func (q *MatchingQueue) Len() int {
	return q.queue.Len()
}

// Enqueue adds a ticket to the queue.
func (q *MatchingQueue) Enqueue(ticket schema.Ticket) {
	q.queue.Push(ticket)
}

// Dequeue removes and returns the oldest ticket from the queue.
func (q *MatchingQueue) Dequeue() (schema.Ticket, bool) {
	if q.Len() == 0 {
		return schema.Ticket{}, false // Return an empty ticket if the queue is empty
	}

	tkt, ok := q.queue.Pop().(schema.Ticket)
	return tkt, ok
}
