package matchmaking

import "testing"

func TestQueuePairsInOrder(t *testing.T) {
	q := NewQueue(nil)
	if _, ok := q.Join(Ticket{UserID: "a", Hero: "death"}); ok {
		t.Fatal("nobody to pair with")
	}
	if _, ok := q.Join(Ticket{UserID: "a", Hero: "strength"}); ok || q.Len() != 1 {
		t.Fatal("re-joining replaces the ticket and never pairs a player with themselves")
	}
	q.Join(Ticket{UserID: "b"})
	// a and b pair as soon as b joins.
	if q.Len() != 0 {
		t.Fatalf("len %d", q.Len())
	}
	q.Join(Ticket{UserID: "c"})
	q.Join(Ticket{UserID: "d"})
	if q.Len() != 0 {
		t.Fatal("c and d pair")
	}
	q.Join(Ticket{UserID: "e"})
	if !q.Leave("e") || q.Leave("e") || q.Len() != 0 {
		t.Fatal("leave")
	}
}

func TestQueueReturnsTheWaitingOpponent(t *testing.T) {
	q := NewQueue(nil)
	q.Join(Ticket{UserID: "a", Hero: "strength"})
	opp, ok := q.Join(Ticket{UserID: "b", Hero: "death"})
	if !ok || opp.UserID != "a" || opp.Hero != "strength" {
		t.Fatalf("%+v %v", opp, ok)
	}
}

func TestCompatibilityIsPluggable(t *testing.T) {
	q := NewQueue(func(a, b Ticket) bool { return a.UserID != b.UserID && a.Hero == b.Hero })
	q.Join(Ticket{UserID: "a", Hero: "death"})
	if _, ok := q.Join(Ticket{UserID: "b", Hero: "strength"}); ok {
		t.Fatal("incompatible pair")
	}
	if opp, ok := q.Join(Ticket{UserID: "c", Hero: "death"}); !ok || opp.UserID != "a" {
		t.Fatal("compatible pair")
	}
}
