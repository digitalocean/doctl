package agents

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStreamChunk_gatewayWrapper(t *testing.T) {
	raw := `{"result":{"event_id":"evt_1","session_id":"sess_1","token_chunk":{"text":"hi"}}}`
	ev, err := parseStreamChunk([]byte(raw))
	require.NoError(t, err)
	require.Equal(t, "evt_1", ev.EventID)
	require.NotNil(t, ev.TokenChunk)
	require.Equal(t, "hi", ev.TokenChunk.Text)
}

func TestParseStreamFixture_concatenated(t *testing.T) {
	fixture := `{"result":{"event_id":"evt_1","session_id":"s1","token_chunk":{"text":"a"}}}
{"result":{"event_id":"evt_2","session_id":"s1","token_chunk":{"text":"b"}}}`
	events, err := ParseStreamFixture(fixture)
	require.NoError(t, err)
	require.Len(t, events, 2)
}

func TestParseStreamChunk_toolCallCompleted_stringDuration(t *testing.T) {
	// protojson / grpc-gateway encodes int64 as a JSON string.
	raw := `{"result":{"event_id":"evt_1","session_id":"sess_1","tool_call_completed":{"tool_call_id":"tc_1","ok":true,"duration_ms":"4200","summary":"read login.go"}}}`
	ev, err := parseStreamChunk([]byte(raw))
	require.NoError(t, err)
	require.NotNil(t, ev.ToolCallCompleted)
	require.Equal(t, int64(4200), ev.ToolCallCompleted.DurationMs.Int64())
}

func TestParseStreamFixture_noTrailingNewline(t *testing.T) {
	fixture := `{"result":{"event_id":"evt_1","session_id":"s1","token_chunk":{"text":"done"}}}`
	events, err := ParseStreamFixture(fixture)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "done", events[0].TokenChunk.Text)
}
