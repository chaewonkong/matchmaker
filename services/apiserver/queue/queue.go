package queue

import (
	"container/heap"

	"github.com/chaewonkong/matchmaker/schema"
)

type queue []schema.Ticket

// Less implements heap.Interface.
func (q queue) Less(i int, j int) bool {
	return q[i].Timestamp.Before(q[j].Timestamp)
}

// Pop implements heap.Interface.
func (q *queue) Pop() any {
	old := *q
	n := len(old)
	ticket := old[n-1]
	*q = old[0 : n-1]

	return ticket
}

// Push implements heap.Interface.
func (q *queue) Push(x any) {
	ticket, ok := x.(schema.Ticket)
	if !ok {
		return
	}
	*q = append(*q, ticket)
}

// Swap implements heap.Interface.
func (q *queue) Swap(i int, j int) {
	qs := *q
	qs[i], qs[j] = qs[j], qs[i]
}

// Len implements heap.Interface.
func (q queue) Len() int {
	return len(q)
}

var _ heap.Interface = (*queue)(nil)
