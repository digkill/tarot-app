package match

import "sync"

// Registry finds live matches by id and by player. Connections from many
// goroutines use it, so it is locked; the matches themselves are not.
type Registry struct {
	mu     sync.Mutex
	byID   map[string]*Match
	byUser map[string]*Match
}

func NewRegistry() *Registry {
	return &Registry{byID: map[string]*Match{}, byUser: map[string]*Match{}}
}

func (r *Registry) Add(m *Match) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[m.ID] = m
	for _, p := range m.Players {
		r.byUser[p.ID] = m
	}
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.byID[id]
	if m == nil {
		return
	}
	delete(r.byID, id)
	for _, p := range m.Players {
		if r.byUser[p.ID] == m {
			delete(r.byUser, p.ID)
		}
	}
}

func (r *Registry) Get(id string) *Match {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.byID[id]
}

// ForUser is the live match a user is playing, if any.
func (r *Registry) ForUser(userID string) *Match {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.byUser[userID]
}

func (r *Registry) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byID)
}

// All is a snapshot of the live matches.
func (r *Registry) All() []*Match {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Match, 0, len(r.byID))
	for _, m := range r.byID {
		out = append(out, m)
	}
	return out
}
