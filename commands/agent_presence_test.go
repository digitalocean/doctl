package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

func TestHostedAgentsPresenceURL(t *testing.T) {
	tests := []struct {
		name    string
		apiURL  string
		want    string
		wantErr bool
	}{
		{name: "default API", apiURL: "", want: "wss://api.digitalocean.com/v2/agents/sessions/s-1/presence?client=doctl"},
		{name: "preview host", apiURL: "https://ohr-agent.do-ai-test.run", want: "wss://ohr-agent.do-ai-test.run/v2/agents/sessions/s-1/presence?client=doctl"},
		{name: "trailing slash and a base path", apiURL: "https://example.test/api/", want: "wss://example.test/api/v2/agents/sessions/s-1/presence?client=doctl"},
		{name: "plain http for a local stack", apiURL: "http://127.0.0.1:8080", want: "ws://127.0.0.1:8080/v2/agents/sessions/s-1/presence?client=doctl"},
		{name: "unsupported scheme", apiURL: "ftp://example.test", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := viper.GetString("api-url")
			viper.Set("api-url", tt.apiURL)
			defer viper.Set("api-url", prev)
			got, err := hostedAgentsPresenceURL("s-1")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got %q, want an error", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestPresenceRequested(t *testing.T) {
	tests := []struct {
		name string
		flag bool
		env  string
		want bool
	}{
		{name: "off by default", want: false},
		{name: "flag", flag: true, want: true},
		{name: "env", env: "1", want: true},
		{name: "env false", env: "false", want: false},
		{name: "env garbage", env: "yes please", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(presenceEnv, tt.env)
			if got := presenceRequested(tt.flag); got != tt.want {
				t.Fatalf("presenceRequested(%v) with %s=%q = %v, want %v", tt.flag, presenceEnv, tt.env, got, tt.want)
			}
		})
	}
}

func participantFor(conn, id, email string) presenceParticipant {
	return presenceParticipant{ConnID: conn, Actor: runActor{Kind: "user", ID: id, Email: email}, State: "active"}
}

func TestPresenceRosterApply(t *testing.T) {
	now := time.Date(2026, 10, 15, 20, 0, 0, 0, time.UTC)
	soon := now.Add(5 * time.Second)
	me := participantFor("me", "u-me", "me@example.com")
	bob := participantFor("bob-1", "u-bob", "bob@example.com")
	bob2 := participantFor("bob-2", "u-bob", "bob@example.com")
	carol := participantFor("carol-1", "u-carol", "carol@example.com")
	myOther := participantFor("me-2", "u-me", "me@example.com")

	r := newPresenceRoster()
	if got := r.prefix(now); got != "" {
		t.Fatalf("prefix before any snapshot = %q", got)
	}

	steps := []struct {
		name        string
		frame       presenceFrame
		wantNotices []string
		wantResync  bool
		wantPrefix  string
	}{
		{
			name:        "first snapshot introduces the room",
			frame:       presenceFrame{Type: "presence.snapshot", Seq: 3, SelfConnID: "me", Participants: []presenceParticipant{me, bob}},
			wantNotices: []string{"here with bob@example.com"},
			wantPrefix:  "[with bob] ",
		},
		{
			name:        "someone joins",
			frame:       presenceFrame{Type: "presence.joined", Seq: 4, Participant: &carol},
			wantNotices: []string{"carol@example.com joined"},
			wantPrefix:  "[with bob, carol] ",
		},
		{
			name:       "the same person on a second device is not announced",
			frame:      presenceFrame{Type: "presence.joined", Seq: 5, Participant: &bob2},
			wantPrefix: "[with bob, carol] ",
		},
		{
			name:       "my own other device is not someone else",
			frame:      presenceFrame{Type: "presence.joined", Seq: 6, Participant: &myOther},
			wantPrefix: "[with bob, carol] ",
		},
		{
			name:       "typing replaces the roster in the prefix",
			frame:      presenceFrame{Type: "presence.typing", Seq: 7, ConnID: "bob-2", State: "typing", ExpiresAt: &soon},
			wantPrefix: "[bob typing] ",
		},
		{
			name:       "a duplicate frame is ignored",
			frame:      presenceFrame{Type: "presence.left", Seq: 7, ConnID: "carol-1"},
			wantPrefix: "[bob typing] ",
		},
		{
			name:       "stopped clears it",
			frame:      presenceFrame{Type: "presence.typing", Seq: 8, ConnID: "bob-2", State: "stopped"},
			wantPrefix: "[with bob, carol] ",
		},
		{
			name:       "one of two devices leaving is not the person leaving",
			frame:      presenceFrame{Type: "presence.left", Seq: 9, ConnID: "bob-1"},
			wantPrefix: "[with bob, carol] ",
		},
		{
			name:        "a gap asks for a resync but still applies",
			frame:       presenceFrame{Type: "presence.left", Seq: 12, ConnID: "bob-2"},
			wantNotices: []string{"bob@example.com left"},
			wantResync:  true,
			wantPrefix:  "[with carol] ",
		},
		{
			name:        "a resync snapshot reports only the difference",
			frame:       presenceFrame{Type: "presence.snapshot", Seq: 15, SelfConnID: "me", Participants: []presenceParticipant{me, bob}},
			wantNotices: []string{"bob@example.com joined", "carol@example.com left"},
			wantPrefix:  "[with bob] ",
		},
	}
	for _, s := range steps {
		notices, resync := r.apply(s.frame)
		if strings.Join(notices, "|") != strings.Join(s.wantNotices, "|") {
			t.Fatalf("%s: notices = %q, want %q", s.name, notices, s.wantNotices)
		}
		if resync != s.wantResync {
			t.Fatalf("%s: resync = %v, want %v", s.name, resync, s.wantResync)
		}
		if got := r.prefix(now); got != s.wantPrefix {
			t.Fatalf("%s: prefix = %q, want %q", s.name, got, s.wantPrefix)
		}
	}
}

func TestPresenceRosterTypingExpiresWithoutAFrame(t *testing.T) {
	now := time.Date(2026, 10, 15, 20, 0, 0, 0, time.UTC)
	until := now.Add(5 * time.Second)
	r := newPresenceRoster()
	r.apply(presenceFrame{Type: "presence.snapshot", Seq: 1, SelfConnID: "me", Participants: []presenceParticipant{
		participantFor("me", "u-me", "me@example.com"),
		participantFor("bob-1", "u-bob", "bob@example.com"),
	}})
	r.apply(presenceFrame{Type: "presence.typing", Seq: 2, ConnID: "bob-1", State: "typing", ExpiresAt: &until})
	if got := r.prefix(now); got != "[bob typing] " {
		t.Fatalf("prefix = %q", got)
	}
	later := now.Add(6 * time.Second)
	if got := r.prefix(later); got != "[with bob] " {
		t.Fatalf("prefix after expiry = %q, want the indicator gone with no stopped frame", got)
	}
	if !r.dropExpiredTyping(later) {
		t.Fatal("dropExpiredTyping found nothing to drop")
	}
}

func TestPresenceRosterSummarizesALargeRoom(t *testing.T) {
	r := newPresenceRoster()
	r.apply(presenceFrame{Type: "presence.snapshot", Seq: 1, SelfConnID: "me", Participants: []presenceParticipant{
		participantFor("me", "u-me", "me@example.com"),
		participantFor("a", "u-a", "ann@example.com"),
		participantFor("b", "u-b", "ben@example.com"),
		participantFor("c", "u-c", "cat@example.com"),
	}})
	if got := r.prefix(time.Now()); got != "[with ann +2] " {
		t.Fatalf("prefix = %q", got)
	}
}

func TestPresenceTypingIsThrottledAndStoppedOnce(t *testing.T) {
	clock := time.Date(2026, 10, 15, 20, 0, 0, 0, time.UTC)
	p := newPresenceSession("ws://unused", nil, func(string) {}, func() {})
	p.now = func() time.Time { return clock }

	p.noteTyping()
	if len(p.send) != 0 {
		t.Fatal("typing sent while not connected")
	}
	p.setConnected(true)

	drain := func() []string {
		var out []string
		for len(p.send) > 0 {
			out = append(out, string(<-p.send))
		}
		return out
	}
	p.noteTyping()
	p.noteTyping()
	clock = clock.Add(time.Second)
	p.noteTyping()
	if got := drain(); len(got) != 1 || !strings.Contains(got[0], `"start"`) {
		t.Fatalf("frames within the refresh window = %q, want one start", got)
	}
	clock = clock.Add(2 * time.Second)
	p.noteTyping()
	if got := drain(); len(got) != 1 {
		t.Fatalf("frames after the refresh window = %q, want one refresh", got)
	}
	p.stopTyping()
	p.stopTyping()
	if got := drain(); len(got) != 1 || !strings.Contains(got[0], `"stop"`) {
		t.Fatalf("stop frames = %q, want exactly one stop", got)
	}
}

// fakePresenceServer speaks just enough mars.presence.v1 to drive the client:
// a snapshot with bob in it, then bob joining on a second socket's behalf,
// and it records what the client sends.
func fakePresenceServer(t *testing.T) (*httptest.Server, *[]string, *sync.Mutex) {
	t.Helper()
	var (
		mu       sync.Mutex
		received []string
	)
	upgrader := websocket.Upgrader{Subprotocols: []string{presenceSubprotocol}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || !strings.HasSuffix(r.URL.Path, "/v2/agents/sessions/s-1/presence") {
			http.Error(w, "nope", http.StatusNotFound)
			return
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		send := func(f presenceFrame) { _ = ws.WriteJSON(f) }
		send(presenceFrame{Type: "presence.snapshot", Seq: 1, SelfConnID: "me", Participants: []presenceParticipant{
			participantFor("me", "u-me", "me@example.com"),
			participantFor("bob-1", "u-bob", "bob@example.com"),
		}})
		carol := participantFor("carol-1", "u-carol", "carol@example.com")
		send(presenceFrame{Type: "presence.joined", Seq: 2, Participant: &carol})
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			mu.Lock()
			received = append(received, string(data))
			mu.Unlock()
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &received, &mu
}

func TestPresenceSessionReceivesTheRosterAndSendsTyping(t *testing.T) {
	srv, received, mu := fakePresenceServer(t)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/v2/agents/sessions/s-1/presence?client=doctl"

	var (
		nmu     sync.Mutex
		notices []string
	)
	redraws := make(chan struct{}, 16)
	p := newPresenceSession(wsURL, tunnelHeader("tok", "dev-1"), func(n string) {
		nmu.Lock()
		notices = append(notices, n)
		nmu.Unlock()
	}, func() {
		select {
		case redraws <- struct{}{}:
		default:
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.run(ctx)

	deadline := time.After(5 * time.Second)
	for {
		if strings.HasPrefix(p.roster.prefix(time.Now()), "[with bob, carol]") {
			break
		}
		select {
		case <-redraws:
		case <-deadline:
			t.Fatalf("roster never reached bob and carol: prefix %q", p.roster.prefix(time.Now()))
		}
	}
	nmu.Lock()
	got := strings.Join(notices, "|")
	nmu.Unlock()
	if got != "here with bob@example.com|carol@example.com joined" {
		t.Fatalf("notices = %q", got)
	}

	p.noteTyping()
	p.stopTyping()
	for i := 0; i < 50; i++ {
		mu.Lock()
		n := len(*received)
		mu.Unlock()
		if n >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*received) < 2 {
		t.Fatalf("server received %q, want a start and a stop", *received)
	}
	var first, second map[string]string
	_ = json.Unmarshal([]byte((*received)[0]), &first)
	_ = json.Unmarshal([]byte((*received)[1]), &second)
	if first["type"] != "typing" || first["state"] != "start" || second["state"] != "stop" {
		t.Fatalf("server received %q", *received)
	}
}
