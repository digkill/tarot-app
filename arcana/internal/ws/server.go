// Package ws is the WebSocket transport: it authenticates connections, keeps
// one session per user, runs matchmaking and routes intents to live matches.
// It holds no game rules.
package ws

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/digkill/tarot-app/arcana/internal/auth"
	"github.com/digkill/tarot-app/arcana/internal/game"
	"github.com/digkill/tarot-app/arcana/internal/heroes"
	"github.com/digkill/tarot-app/arcana/internal/match"
	"github.com/digkill/tarot-app/arcana/internal/matchmaking"
	"github.com/digkill/tarot-app/arcana/internal/protocol"
	"github.com/digkill/tarot-app/arcana/internal/storage"
)

type Config struct {
	// A ping every PingInterval; no pong within PingTimeout drops the
	// connection. Silence between game actions is never a disconnect.
	PingInterval time.Duration
	PingTimeout  time.Duration
	HelloTimeout time.Duration
	WriteTimeout time.Duration
	MaxMessage   int64
	// Frames per second a client may send before being disconnected.
	RateLimit int
	Match     match.Config
}

func DefaultConfig() Config {
	return Config{
		PingInterval: 15 * time.Second, PingTimeout: 10 * time.Second, HelloTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, MaxMessage: 8 << 10, RateLimit: 20, Match: match.DefaultConfig(),
	}
}

type Server struct {
	cfg      Config
	verifier *auth.Verifier
	queue    *matchmaking.Queue
	matches  *match.Registry
	store    storage.Store

	mu       sync.Mutex
	sessions map[string]*session
}

func NewServer(cfg Config, verifier *auth.Verifier, store storage.Store) *Server {
	return &Server{
		cfg: cfg, verifier: verifier, store: store,
		queue: matchmaking.NewQueue(nil), matches: match.NewRegistry(),
		sessions: map[string]*session{},
	}
}

func (s *Server) Matches() *match.Registry { return s.matches }

// Shutdown aborts every live match (recorded as server_error) and waits for
// them to be saved.
func (s *Server) Shutdown(ctx context.Context) {
	live := s.matches.All()
	for _, m := range live {
		m.Abort()
	}
	for _, m := range live {
		select {
		case <-m.Done():
		case <-ctx.Done():
			return
		}
	}
}

type session struct {
	userID string
	conn   *websocket.Conn
	out    chan protocol.ServerMessage
	ctx    context.Context
	cancel context.CancelCauseFunc
	// The match this connection receives frames for. Set by this
	// connection and by the opponent's when a match is created.
	mu       sync.Mutex
	attached string
}

func (c *session) attachedTo() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.attached
}

func (c *session) attach(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attached = id
}

// Send queues a frame. A client too slow to drain its queue is dropped
// rather than allowed to stall the match.
func (c *session) Send(m protocol.ServerMessage) {
	select {
	case c.out <- m:
	case <-c.ctx.Done():
	default:
		c.cancel(errors.New("send queue full"))
	}
}

func (c *session) error(code, message, ref string) {
	c.Send(protocol.ServerMessage{Type: protocol.Error, Payload: protocol.ErrorPayload{Code: code, Message: message, Ref: ref}})
}

var errReplaced = errors.New("replaced by a newer connection")

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Native apps send no Origin; browsers from other sites are refused by
	// the library's same-origin check.
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(s.cfg.MaxMessage)
	ctx, cancel := context.WithCancelCause(r.Context())
	defer cancel(nil)

	userID, err := s.hello(ctx, conn)
	if err != nil {
		conn.Close(websocket.StatusPolicyViolation, err.Error())
		return
	}
	sess := &session{userID: userID, conn: conn, out: make(chan protocol.ServerMessage, 64), ctx: ctx, cancel: cancel}
	s.register(sess)
	defer s.unregister(sess)

	go s.writeLoop(sess)
	go s.pingLoop(sess)

	active := ""
	if m := s.matches.ForUser(userID); m != nil {
		active = m.ID
	}
	sess.Send(protocol.ServerMessage{Type: protocol.Hello, Payload: protocol.HelloPayload{
		UserID: userID, Protocol: protocol.Version, RulesVersion: game.RulesVersion,
		ActiveMatch: active, Emotes: protocol.Emotes,
	}})
	s.readLoop(sess)

	status, reason := websocket.StatusNormalClosure, ""
	if cause := context.Cause(ctx); cause != nil && !errors.Is(cause, context.Canceled) {
		status, reason = websocket.StatusPolicyViolation, cause.Error()
		if errors.Is(cause, errReplaced) {
			status = 4001
		}
	}
	conn.Close(status, reason)
}

func (s *Server) hello(ctx context.Context, conn *websocket.Conn) (string, error) {
	hctx, cancel := context.WithTimeout(ctx, s.cfg.HelloTimeout)
	defer cancel()
	var msg protocol.ClientMessage
	if err := wsjson.Read(hctx, conn, &msg); err != nil {
		return "", fmt.Errorf("hello expected")
	}
	if msg.Type != protocol.Hello {
		return "", fmt.Errorf("hello expected")
	}
	if msg.Protocol != protocol.Version {
		return "", fmt.Errorf("unsupported protocol %d", msg.Protocol)
	}
	userID, err := s.verifier.UserID(msg.Token)
	if err != nil {
		return "", fmt.Errorf("unauthorized")
	}
	return userID, nil
}

func (s *Server) register(sess *session) {
	s.mu.Lock()
	old := s.sessions[sess.userID]
	s.sessions[sess.userID] = sess
	s.mu.Unlock()
	if old != nil {
		old.cancel(errReplaced)
	}
}

func (s *Server) unregister(sess *session) {
	s.mu.Lock()
	current := s.sessions[sess.userID] == sess
	if current {
		delete(s.sessions, sess.userID)
	}
	s.mu.Unlock()
	if current {
		s.queue.Leave(sess.userID)
	}
	if m := s.matches.ForUser(sess.userID); m != nil {
		m.Detach(sess.userID, sess)
	}
}

func (s *Server) session(userID string) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[userID]
}

func (s *Server) writeLoop(sess *session) {
	for {
		select {
		case <-sess.ctx.Done():
			return
		case msg := <-sess.out:
			wctx, cancel := context.WithTimeout(sess.ctx, s.cfg.WriteTimeout)
			err := wsjson.Write(wctx, sess.conn, msg)
			cancel()
			if err != nil {
				sess.cancel(err)
				return
			}
		}
	}
}

func (s *Server) pingLoop(sess *session) {
	ticker := time.NewTicker(s.cfg.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-sess.ctx.Done():
			return
		case <-ticker.C:
			pctx, cancel := context.WithTimeout(sess.ctx, s.cfg.PingTimeout)
			err := sess.conn.Ping(pctx)
			cancel()
			if err != nil {
				sess.cancel(fmt.Errorf("ping: %w", err))
				return
			}
		}
	}
}

func (s *Server) readLoop(sess *session) {
	windowStart, count := time.Now(), 0
	for {
		// Read raw frames: wsjson closes the socket on malformed JSON, which
		// deserves an answer instead.
		typ, data, err := sess.conn.Read(sess.ctx)
		if err != nil {
			return
		}
		var msg protocol.ClientMessage
		if typ != websocket.MessageText || json.Unmarshal(data, &msg) != nil {
			sess.error("bad_request", "malformed message", "")
			continue
		}
		if now := time.Now(); now.Sub(windowStart) >= time.Second {
			windowStart, count = now, 0
		}
		count++
		if count > s.cfg.RateLimit {
			sess.cancel(errors.New("rate limit exceeded"))
			return
		}
		s.dispatch(sess, msg)
	}
}

func (s *Server) dispatch(sess *session, msg protocol.ClientMessage) {
	switch msg.Type {
	case protocol.QueueJoin:
		s.joinQueue(sess, msg)
	case protocol.QueueLeave:
		s.queue.Leave(sess.userID)
		sess.Send(protocol.ServerMessage{Type: protocol.QueueLeft})
	case protocol.MatchResume:
		m := s.findMatch(sess, msg)
		if m == nil {
			return
		}
		if m.Attach(sess.userID, sess) {
			sess.attach(m.ID)
		} else {
			sess.error("match_finished", "the match is over", msg.Ref)
		}
	case protocol.Mulligan, protocol.CardPlay, protocol.HeroAbility, protocol.HeroUltimate,
		protocol.TurnEnd, protocol.Surrender, protocol.PlayerEmote:
		m := s.findMatch(sess, msg)
		if m == nil {
			return
		}
		if sess.attachedTo() != m.ID {
			m.Attach(sess.userID, sess)
			sess.attach(m.ID)
		}
		if !m.Submit(sess.userID, msg) {
			sess.error("match_finished", "the match is over", msg.Ref)
		}
	case protocol.Hello:
		sess.error("already_authenticated", "hello was already sent", msg.Ref)
	default:
		sess.error("unknown_type", "unknown message type", msg.Ref)
	}
}

// findMatch checks that the match exists and that this user plays in it.
func (s *Server) findMatch(sess *session, msg protocol.ClientMessage) *match.Match {
	m := s.matches.Get(msg.MatchID)
	if msg.MatchID == "" && msg.Type == protocol.MatchResume {
		m = s.matches.ForUser(sess.userID)
	}
	if m == nil {
		sess.error("match_not_found", "no such live match", msg.Ref)
		return nil
	}
	if !m.Has(sess.userID) {
		sess.error("not_in_match", "you are not in this match", msg.Ref)
		return nil
	}
	return m
}

func (s *Server) joinQueue(sess *session, msg protocol.ClientMessage) {
	if m := s.matches.ForUser(sess.userID); m != nil {
		sess.Send(protocol.ServerMessage{Type: protocol.Error, MatchID: m.ID, Payload: protocol.ErrorPayload{
			Code: "already_in_match", Message: "finish or resume the current match", Ref: msg.Ref}})
		return
	}
	hero := msg.Hero
	if hero == "" {
		all := heroes.All()
		hero = all[int(randomUint64()%uint64(len(all)))].ID
	}
	if _, ok := heroes.ByID(hero); !ok {
		sess.error("invalid_hero", "unknown hero", msg.Ref)
		return
	}
	ticket := matchmaking.Ticket{UserID: sess.userID, Hero: hero, JoinedAt: time.Now()}
	opponent, paired := s.queue.Join(ticket)
	if !paired {
		sess.Send(protocol.ServerMessage{Type: protocol.QueueWaiting, Payload: map[string]string{"hero": hero}})
		return
	}
	s.startMatch(opponent, ticket)
}

func (s *Server) startMatch(a, b matchmaking.Ticket) {
	var outs [2]match.Outbox
	sessions := [2]*session{s.session(a.UserID), s.session(b.UserID)}
	for i, sess := range sessions {
		if sess != nil {
			outs[i] = sess
		}
	}
	id := newUUID()
	m, err := match.New(id, randomUint64(),
		game.PlayerSetup{ID: a.UserID, Hero: a.Hero}, game.PlayerSetup{ID: b.UserID, Hero: b.Hero},
		outs, s.cfg.Match, s.finished)
	if err != nil {
		slog.Error("arcana: create match", "err", err)
		for _, sess := range sessions {
			if sess != nil {
				sess.error("server_error", "could not start the match", "")
			}
		}
		return
	}
	s.matches.Add(m)
	for _, sess := range sessions {
		if sess != nil {
			sess.attach(id)
		}
	}
	m.Run()
}

// finished runs in the match goroutine when a match ends.
func (s *Server) finished(r match.Result) {
	s.matches.Remove(r.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	record := storage.MatchRecord{
		ID: r.ID, Player1ID: r.Players[0].ID, Player2ID: r.Players[1].ID,
		Player1Hero: r.Players[0].Hero, Player2Hero: r.Players[1].Hero, WinnerID: r.WinnerID,
		Reason: r.Reason, Seed: r.Seed, RulesVersion: r.Rules.Version, ProtocolVersion: protocol.Version,
		Turns: r.Turns, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt, Replay: r.Log,
	}
	if err := s.store.SaveMatch(ctx, record); err != nil {
		slog.Error("arcana: save match", "match", r.ID, "err", err)
	}
}

func randomUint64() uint64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return binary.LittleEndian.Uint64(b[:])
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
