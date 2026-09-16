package game

import "fmt"

// Entry is one committed action with everything it caused.
type Entry struct {
	Seq      uint64   `json:"seq"`
	Player   string   `json:"player,omitempty"`
	Action   string   `json:"action"`
	CardUID  string   `json:"card_uid,omitempty"`
	CardID   string   `json:"card_id,omitempty"`
	Reversed bool     `json:"reversed,omitempty"`
	Target   *Target  `json:"target,omitempty"`
	CardUIDs []string `json:"card_uids,omitempty"`
	Reason   Reason   `json:"reason,omitempty"`
	Hidden   bool     `json:"hidden,omitempty"`
	Events   []Event  `json:"events,omitempty"`

	// Only the acting player may see the details (a mulligan).
	private bool
}

// Event is an outcome of an action, for the client to animate.
type Event struct {
	Kind     string `json:"kind"`
	Player   string `json:"player,omitempty"`
	Amount   int    `json:"amount,omitempty"`
	Absorbed int    `json:"absorbed,omitempty"`
	Crit     bool   `json:"crit,omitempty"`
	Element  string `json:"element,omitempty"`
	Status   string `json:"status,omitempty"`
	CardID   string `json:"card_id,omitempty"`
	CardUID  string `json:"card_uid,omitempty"`
	Reason   string `json:"reason,omitempty"`

	// Card or status details only the affected player may see.
	private bool
}

func (e *Entry) add(ev Event) {
	if e != nil {
		e.Events = append(e.Events, ev)
	}
}

// redacted is the entry as a given player may see it.
func (e Entry) redacted(viewer string) Entry {
	out := e
	if e.Player != viewer {
		out.CardUIDs = nil
		if e.Hidden {
			out.CardUID, out.CardID, out.Reversed, out.Target = "", "", false, nil
		}
	}
	out.Events = make([]Event, 0, len(e.Events))
	for _, ev := range e.Events {
		if ev.Player != viewer && ev.private {
			if ev.Status != "" {
				continue // a hidden status is not even hinted at
			}
			ev.CardID, ev.CardUID = "", ""
		}
		// Everything a veiled card did to its own player stays private too.
		if e.Hidden && e.Player != viewer && ev.Kind == "repeat" {
			ev.CardID = ""
		}
		out.Events = append(out.Events, ev)
	}
	return out
}

// RedactLog is a stored log as one player may see it. Stored entries lose the
// in-memory privacy flags, so draws and mulligans are hidden by kind here.
func RedactLog(log []Entry, viewer string) []Entry {
	out := make([]Entry, 0, len(log))
	for _, e := range log {
		if e.Action == ActMulligan {
			e.private = true
		}
		events := make([]Event, len(e.Events))
		for i, ev := range e.Events {
			switch ev.Kind {
			case "draw", "card_added":
				ev.private = true
			case "status_added", "status_removed":
				ev.private = statusDefs[ev.Status].Hidden
			}
			events[i] = ev
		}
		e.Events = events
		out = append(out, e.redacted(viewer))
	}
	return out
}

// Replay rebuilds a match from its seed, players and recorded actions,
// failing if any action no longer applies — the check that a stored replay
// matches the rules it claims.
func Replay(rules Rules, seed uint64, a, b PlayerSetup, log []Entry) (*Game, error) {
	g, err := NewWithRules(rules, seed, a, b)
	if err != nil {
		return nil, err
	}
	for _, e := range log {
		target := Target{}
		if e.Target != nil {
			target = *e.Target
		}
		switch e.Action {
		case ActMulligan:
			err = g.Mulligan(e.Player, e.CardUIDs)
		case ActMulliganTimeout:
			err = g.SkipMulligans()
		case ActPlayCard:
			err = g.PlayCard(e.Player, e.CardUID, target)
		case ActAbility:
			err = g.UseAbility(e.Player, target)
		case ActUltimate:
			err = g.UseUltimate(e.Player, target)
		case ActEndTurn:
			err = g.EndTurn(e.Player)
		case ActTurnTimeout:
			err = g.TimeoutTurn()
		case ActSurrender, ActForfeit:
			err = g.Forfeit(e.Player, e.Reason)
		case ActAbort:
			err = g.Abort()
		default:
			err = fmt.Errorf("unknown action %q", e.Action)
		}
		if err != nil {
			return nil, fmt.Errorf("replay seq %d (%s): %w", e.Seq, e.Action, err)
		}
		if g.Seq != e.Seq {
			return nil, fmt.Errorf("replay seq %d: got %d", e.Seq, g.Seq)
		}
	}
	return g, nil
}

// Setup reports the players a game was created with, for storing replays.
func (g *Game) Setup() (PlayerSetup, PlayerSetup) { return g.setup[0], g.setup[1] }
