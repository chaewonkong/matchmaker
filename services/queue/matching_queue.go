package queue

import (
	"container/heap"

	"github.com/chaewonkong/matchmaker/schema"
)

// MatchingQueue is a priority queue for matchmaking tickets.
type MatchingQueue struct {
	queue queue
	index map[string]*ticketEntry
}

// New creates a new MatchingQueue instance.
func New() *MatchingQueue {
	return &MatchingQueue{
		queue: queue{},
		index: make(map[string]*ticketEntry),
	}
}

// Len returns the number of tickets in the queue.
func (q *MatchingQueue) Len() int {
	return q.queue.Len()
}

// Enqueue adds a ticket to the queue.
func (q *MatchingQueue) Enqueue(ticket schema.Ticket) {
	if _, exists := q.index[ticket.ID]; exists {
		return // Ticket already exists in the queue
	}
	entry := &ticketEntry{
		Ticket: ticket,
	}
	heap.Push(&q.queue, entry)
	q.index[ticket.ID] = entry
}

// Dequeue removes and returns the oldest ticket from the queue.
func (q *MatchingQueue) Dequeue() (schema.Ticket, bool) {
	if q.Len() == 0 {
		return schema.Ticket{}, false // Return an empty ticket if the queue is empty
	}

	entry, ok := heap.Pop(&q.queue).(*ticketEntry)
	delete(q.index, entry.Ticket.ID)
	return entry.Ticket, ok
}

// RemoveTicketByID removes a ticket from the queue by its ID.
func (q *MatchingQueue) RemoveTicketByID(ticketID string) (schema.Ticket, bool) {
	entry, exists := q.index[ticketID]
	if !exists {
		return schema.Ticket{}, false // Ticket not found
	}

	v := heap.Remove(&q.queue, entry.index)
	tkt := v.(*ticketEntry).Ticket
	delete(q.index, tkt.ID)

	return tkt, true
}
