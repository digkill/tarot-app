// Package protocol defines the Arcana Clash WebSocket messages.
//
// Every frame is one JSON object. Clients send intents only — which card, which
// target — and never numbers the server computes. Server frames always carry
// `type`, `seq` (the match state sequence, 0 outside a match) and, for match
// events, `match_id`.
package protocol

import (
	"encoding/json"

	"github.com/digkill/tarot-app/arcana/internal/game"
)

// Version is bumped on any incompatible change to these messages.
const Version = 1

// Client → server.
const (
	Hello        = "hello"
	QueueJoin    = "queue.join"
	QueueLeave   = "queue.leave"
	MatchResume  = "match.resume"
	Mulligan     = "mulligan"
	CardPlay     = "card.play"
	HeroAbility  = "hero.ability"
	HeroUltimate = "hero.ultimate"
	TurnEnd      = "turn.end"
	Surrender    = "match.surrender"
	PlayerEmote  = "player.emote"
)

// Server → client.
const (
	QueueWaiting         = "queue.waiting"
	QueueLeft            = "queue.left"
	MatchStarted         = "match.started"
	MatchState           = "match.state"
	MatchFinished        = "match.finished"
	OpponentDisconnected = "opponent.disconnected"
	OpponentReconnected  = "opponent.reconnected"
	Error                = "error"
)

// ClientMessage is any client frame; each type reads its own fields.
type ClientMessage struct {
	Type     string       `json:"type"`
	Ref      string       `json:"ref,omitempty"` // echoed in an error reply
	Protocol int          `json:"protocol,omitempty"`
	Token    string       `json:"token,omitempty"`
	MatchID  string       `json:"match_id,omitempty"`
	Hero     string       `json:"hero,omitempty"`
	CardUID  string       `json:"card_uid,omitempty"`
	CardUIDs []string     `json:"card_uids,omitempty"`
	Target   *game.Target `json:"target,omitempty"`
	Emote    string       `json:"emote,omitempty"`
}

// ServerMessage is any server frame.
type ServerMessage struct {
	Type    string `json:"type"`
	Seq     uint64 `json:"seq"`
	MatchID string `json:"match_id,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

type HelloPayload struct {
	UserID       string   `json:"user_id"`
	Protocol     int      `json:"protocol"`
	RulesVersion int      `json:"rules_version"`
	ActiveMatch  string   `json:"active_match,omitempty"`
	Emotes       []string `json:"emotes"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Ref     string `json:"ref,omitempty"`
}

type PlayerInfo struct {
	ID   string `json:"id"`
	Hero string `json:"hero"`
}

type StartedPayload struct {
	You      PlayerInfo `json:"you"`
	Opponent PlayerInfo `json:"opponent"`
}

type StatePayload struct {
	State             game.View `json:"state"`
	ServerTime        int64     `json:"server_time"` // unix ms
	DeadlineAt        int64     `json:"deadline_at"` // unix ms: end of the turn or mulligan
	OpponentConnected bool      `json:"opponent_connected"`
}

type DisconnectedPayload struct {
	ReconnectBy int64 `json:"reconnect_by"` // unix ms
}

type EmotePayload struct {
	Player string `json:"player"`
	Emote  string `json:"emote"`
}

type FinishedPayload struct {
	Winner string      `json:"winner,omitempty"`
	Reason game.Reason `json:"reason"`
	Result string      `json:"result"` // win, loss or none, for the receiving player
	State  game.View   `json:"state"`
}

// Emotes are a fixed list: free text between strangers needs moderation.
var Emotes = []string{"greetings", "well_played", "wow", "oops", "thinking", "good_game"}

func ValidEmote(e string) bool {
	for _, x := range Emotes {
		if x == e {
			return true
		}
	}
	return false
}

func Decode(data []byte) (ClientMessage, error) {
	var m ClientMessage
	err := json.Unmarshal(data, &m)
	return m, err
}
