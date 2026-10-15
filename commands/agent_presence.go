package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

// Session presence (MARSOHS-2247): an opt-in WebSocket beside the attach event
// stream that says who else has the session open and who is typing. The wire
// contract is mars.presence.v1; this file is its doctl client.
const (
	presenceSubprotocol = "mars.presence.v1"
	// presenceEnv opts into presence without the hidden --presence flag.
	presenceEnv = "DOCTL_AGENTS_PRESENCE"
	// presenceTypingRefresh is how often a still-typing client re-sends start;
	// the server holds the indicator 5s past the last one.
	presenceTypingRefresh = 2 * time.Second
	presenceHealthy       = 30 * time.Second
	presenceMaxBackoff    = 30 * time.Second
)

// presenceRequested reports whether attach should open the presence socket.
func presenceRequested(flag bool) bool {
	if flag {
		return true
	}
	v, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(presenceEnv)))
	return err == nil && v
}

// hostedAgentsPresenceURL builds the presence WebSocket URL from the same API
// base URL the HTTP client uses, so --api-url / DIGITALOCEAN_API_URL (e.g. the
// preview host) is honored and no host is assumed.
func hostedAgentsPresenceURL(sessionID string) (string, error) {
	raw := viper.GetString("api-url")
	if raw == "" {
		raw = defaultAPIURL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid API URL %q: %w", raw, err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported API URL scheme %q", u.Scheme)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/v2/agents/sessions/" + url.PathEscape(sessionID) + "/presence"
	u.RawQuery = url.Values{"client": []string{"doctl"}}.Encode()
	return u.String(), nil
}

type presenceParticipant struct {
	ConnID      string     `json:"conn_id"`
	Actor       runActor   `json:"actor"`
	State       string     `json:"state"`
	TypingUntil *time.Time `json:"typing_until,omitempty"`
}

type presenceFrame struct {
	Type         string                `json:"type"`
	Seq          int64                 `json:"seq"`
	SelfConnID   string                `json:"self_conn_id"`
	Participants []presenceParticipant `json:"participants"`
	Participant  *presenceParticipant  `json:"participant"`
	ConnID       string                `json:"conn_id"`
	State        string                `json:"state"`
	ExpiresAt    *time.Time            `json:"expires_at"`
}

// presenceRoster is the client's view of who is in the session, reduced from
// the server's frames.
type presenceRoster struct {
	mu      sync.Mutex
	self    string
	seq     int64
	members map[string]presenceParticipant
	typing  map[string]time.Time
}

func newPresenceRoster() *presenceRoster {
	return &presenceRoster{members: map[string]presenceParticipant{}, typing: map[string]time.Time{}}
}

// apply folds one frame in. It returns lines worth telling the user (someone
// arrived or left) and whether the client missed frames and should ask for a
// fresh snapshot.
func (r *presenceRoster) apply(f presenceFrame) (notices []string, resync bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if f.Type == "presence.snapshot" {
		before := r.othersLocked()
		r.self, r.seq = f.SelfConnID, f.Seq
		r.members = map[string]presenceParticipant{}
		r.typing = map[string]time.Time{}
		for _, p := range f.Participants {
			r.members[p.ConnID] = p
			if p.TypingUntil != nil {
				r.typing[p.ConnID] = *p.TypingUntil
			}
		}
		// The first snapshot introduces the room; a resync snapshot only
		// reports the difference, so a reconnect does not re-announce everyone.
		after := r.othersLocked()
		if before == nil {
			if len(after) > 0 {
				notices = append(notices, "here with "+strings.Join(after, ", "))
			}
			return notices, false
		}
		for _, who := range diff(after, before) {
			notices = append(notices, who+" joined")
		}
		for _, who := range diff(before, after) {
			notices = append(notices, who+" left")
		}
		return notices, false
	}
	if f.Seq != 0 {
		if f.Seq <= r.seq {
			return nil, false
		}
		resync = r.seq != 0 && f.Seq > r.seq+1
		r.seq = f.Seq
	}
	switch f.Type {
	case "presence.joined":
		if f.Participant == nil {
			break
		}
		who := f.Participant.Actor.label()
		known := r.hasActorLocked(f.Participant.Actor)
		r.members[f.Participant.ConnID] = *f.Participant
		if f.Participant.ConnID != r.self && !known && !r.isSelfActorLocked(f.Participant.Actor) {
			notices = append(notices, who+" joined")
		}
	case "presence.left":
		p, ok := r.members[f.ConnID]
		delete(r.members, f.ConnID)
		delete(r.typing, f.ConnID)
		if ok && f.ConnID != r.self && !r.hasActorLocked(p.Actor) && !r.isSelfActorLocked(p.Actor) {
			notices = append(notices, p.Actor.label()+" left")
		}
	case "presence.typing":
		if f.State == "typing" && f.ExpiresAt != nil {
			r.typing[f.ConnID] = *f.ExpiresAt
		} else {
			delete(r.typing, f.ConnID)
		}
	case "presence.updated":
		if p, ok := r.members[f.ConnID]; ok {
			p.State = f.State
			r.members[f.ConnID] = p
		}
	}
	return notices, resync
}

// prefix is what attach puts in front of its prompt: who is typing, else who
// else is here, else nothing.
func (r *presenceRoster) prefix(now time.Time) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.members == nil {
		return ""
	}
	var typists []string
	seen := map[string]bool{}
	for conn, until := range r.typing {
		p, ok := r.members[conn]
		if !ok || conn == r.self || r.isSelfActorLocked(p.Actor) || !until.After(now) {
			continue
		}
		if name := shortName(p.Actor); !seen[name] {
			seen[name] = true
			typists = append(typists, name)
		}
	}
	sort.Strings(typists)
	if len(typists) > 0 {
		return "[" + strings.Join(typists, ", ") + " typing] "
	}
	others := r.othersLocked()
	switch {
	case len(others) == 0:
		return ""
	case len(others) <= 2:
		return "[with " + strings.Join(shortNames(others), ", ") + "] "
	default:
		return fmt.Sprintf("[with %s +%d] ", shortNames(others)[0], len(others)-1)
	}
}

// dropExpiredTyping clears lapsed indicators and reports whether any were, so
// the prompt can be repainted without waiting for the next frame.
func (r *presenceRoster) dropExpiredTyping(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	dropped := false
	for conn, until := range r.typing {
		if !until.After(now) {
			delete(r.typing, conn)
			dropped = true
		}
	}
	return dropped
}

// othersLocked lists everyone present but the caller, one entry per person.
// It is nil before the first snapshot, which apply uses to tell the first
// snapshot from a resync.
func (r *presenceRoster) othersLocked() []string {
	if r.self == "" {
		return nil
	}
	set := map[string]bool{}
	for conn, p := range r.members {
		if conn == r.self || r.isSelfActorLocked(p.Actor) {
			continue
		}
		set[p.Actor.label()] = true
	}
	out := make([]string, 0, len(set))
	for who := range set {
		out = append(out, who)
	}
	sort.Strings(out)
	return out
}

func (r *presenceRoster) isSelfActorLocked(a runActor) bool {
	me, ok := r.members[r.self]
	return ok && a.ID != "" && a.ID == me.Actor.ID
}

// hasActorLocked reports whether the person is still present on any socket,
// so a second device joining or leaving is not announced as the person.
func (r *presenceRoster) hasActorLocked(a runActor) bool {
	for _, p := range r.members {
		if p.Actor.label() == a.label() {
			return true
		}
	}
	return false
}

func diff(a, b []string) []string {
	in := map[string]bool{}
	for _, v := range b {
		in[v] = true
	}
	var out []string
	for _, v := range a {
		if !in[v] {
			out = append(out, v)
		}
	}
	return out
}

// shortName keeps a prompt prefix short: an email's local part, else the
// label as-is.
func shortName(a runActor) string { return shortLabel(a.label()) }

func shortLabel(label string) string {
	if i := strings.IndexByte(label, '@'); i > 0 {
		return label[:i]
	}
	return label
}

func shortNames(labels []string) []string {
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = shortLabel(l)
	}
	return out
}

// presenceSession owns the socket: it reconnects, reduces frames into the
// roster, and sends this user's typing signals.
type presenceSession struct {
	url    string
	header http.Header
	roster *presenceRoster
	// notify prints a roster change; redraw repaints the prompt. Both are the
	// attach display's, so presence output never clobbers the input line.
	notify func(string)
	redraw func()
	now    func() time.Time

	send chan []byte

	mu           sync.Mutex
	connected    bool
	typing       bool
	typingSentAt time.Time
}

func newPresenceSession(wsURL string, header http.Header, notify func(string), redraw func()) *presenceSession {
	return &presenceSession{
		url:    wsURL,
		header: header,
		roster: newPresenceRoster(),
		notify: notify,
		redraw: redraw,
		now:    time.Now,
		send:   make(chan []byte, 16),
	}
}

// noteTyping is called on every edit that adds input. It sends start on the
// first keystroke and again at most every presenceTypingRefresh.
func (p *presenceSession) noteTyping() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.connected {
		return
	}
	now := p.now()
	if p.typing && now.Sub(p.typingSentAt) < presenceTypingRefresh {
		return
	}
	if p.enqueueLocked(`{"type":"typing","state":"start"}`) {
		p.typing, p.typingSentAt = true, now
	}
}

// stopTyping is called when the input is submitted or cleared.
func (p *presenceSession) stopTyping() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.typing {
		return
	}
	p.typing = false
	p.enqueueLocked(`{"type":"typing","state":"stop"}`)
}

func (p *presenceSession) enqueueLocked(frame string) bool {
	select {
	case p.send <- []byte(frame):
		return true
	default:
		return false
	}
}

func (p *presenceSession) setConnected(on bool) {
	p.mu.Lock()
	p.connected = on
	if !on {
		p.typing = false
	}
	p.mu.Unlock()
}

// run keeps the socket open until ctx ends, reconnecting with backoff. A
// policy close (1008) means the server will keep refusing; it stops there.
func (p *presenceSession) run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		started := p.now()
		err := p.connectOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		var ce *websocket.CloseError
		if errors.As(err, &ce) && ce.Code == websocket.ClosePolicyViolation {
			p.notify("presence stopped: " + ce.Text)
			return
		}
		if p.now().Sub(started) > presenceHealthy {
			backoff = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > presenceMaxBackoff {
			backoff = presenceMaxBackoff
		}
	}
}

func (p *presenceSession) connectOnce(ctx context.Context) error {
	d := websocket.Dialer{Subprotocols: []string{presenceSubprotocol}, HandshakeTimeout: 15 * time.Second, Proxy: http.ProxyFromEnvironment}
	ws, _, err := d.DialContext(ctx, p.url, p.header)
	if err != nil {
		return err
	}
	defer ws.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
				_ = ws.Close()
				return
			case <-done:
				return
			case frame := <-p.send:
				_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if ws.WriteMessage(websocket.TextMessage, frame) != nil {
					_ = ws.Close()
					return
				}
			case <-tick.C:
				if p.roster.dropExpiredTyping(p.now()) {
					p.redraw()
				}
			}
		}
	}()

	p.setConnected(true)
	defer p.setConnected(false)
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			return err
		}
		var f presenceFrame
		if json.Unmarshal(data, &f) != nil {
			continue
		}
		notices, resync := p.roster.apply(f)
		for _, n := range notices {
			p.notify(n)
		}
		if resync {
			p.mu.Lock()
			p.enqueueLocked(`{"type":"resync"}`)
			p.mu.Unlock()
		}
		p.redraw()
	}
}
