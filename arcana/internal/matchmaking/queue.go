// Package matchmaking pairs players waiting for a match.
//
// The first version is a FIFO queue: the first compatible waiting player is
// the opponent. Compatibility is a single function so rating bands, regions
// or private lobbies can be added without touching the callers.
package matchmaking

import (
	"sync"
	"time"
)

type Ticket struct {
	UserID   string
	Hero     string
	JoinedAt time.Time
}

// Compatible decides whether two tickets may be paired.
type Compatible func(a, b Ticket) bool

func AnyOpponent(a, b Ticket) bool { return a.UserID != b.UserID }

type Queue struct {
	mu         sync.Mutex
	waiting    []Ticket
	compatible Compatible
}

func NewQueue(compatible Compatible) *Queue {
	if compatible == nil {
		compatible = AnyOpponent
	}
	return &Queue{compatible: compatible}
}

// Join pairs the ticket with the longest-waiting compatible player, removing
// both from the queue, or enqueues it. Joining again while queued replaces
// the earlier ticket (e.g. to change hero).
func (q *Queue) Join(t Ticket) (opponent Ticket, paired bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.removeLocked(t.UserID)
	for i, w := range q.waiting {
		if q.compatible(w, t) {
			q.waiting = append(q.waiting[:i:i], q.waiting[i+1:]...)
			return w, true
		}
	}
	q.waiting = append(q.waiting, t)
	return Ticket{}, false
}

func (q *Queue) Leave(userID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.removeLocked(userID)
}

func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.waiting)
}

func (q *Queue) removeLocked(userID string) bool {
	for i, w := range q.waiting {
		if w.UserID == userID {
			q.waiting = append(q.waiting[:i:i], q.waiting[i+1:]...)
			return true
		}
	}
	return false
}
