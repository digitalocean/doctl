package agents

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// streamWrapper is grpc-gateway's JSON envelope for streamed messages.
type streamWrapper struct {
	Result json.RawMessage `json:"result"`
	Error  *streamError    `json:"error"`
}

type streamError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// StreamDecoder reads harness StreamSession HTTP responses (grpc-gateway JSON stream).
type StreamDecoder struct {
	scanner *bufio.Scanner
}

// NewStreamDecoder creates a decoder for a stream response body.
func NewStreamDecoder(r io.Reader) *StreamDecoder {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	sc.Split(splitGatewayJSON)
	return &StreamDecoder{scanner: sc}
}

// Next returns the next Event or io.EOF when done.
func (d *StreamDecoder) Next() (*Event, error) {
	for d.scanner.Scan() {
		line := bytes.TrimSpace(d.scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		ev, err := parseStreamChunk(line)
		if err != nil {
			return nil, err
		}
		if ev != nil {
			return ev, nil
		}
	}
	if err := d.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func parseStreamChunk(data []byte) (*Event, error) {
	// Try grpc-gateway wrapper first.
	var wrap streamWrapper
	if err := json.Unmarshal(data, &wrap); err == nil && len(wrap.Result) > 0 {
		if wrap.Error != nil {
			return nil, fmt.Errorf("stream error: %s", wrap.Error.Message)
		}
		var ev Event
		if err := json.Unmarshal(wrap.Result, &ev); err != nil {
			return nil, err
		}
		return &ev, nil
	}
	var ev Event
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}
	if ev.EventID == "" && ev.SessionID == "" {
		return nil, nil
	}
	return &ev, nil
}

// splitGatewayJSON splits grpc-gateway JSON streams on newline delimiters.
// ForwardResponseStream writes one {"result":...} object per line.
func splitGatewayJSON(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
		line := bytes.TrimSpace(data[:idx])
		if len(line) == 0 {
			return idx + 1, nil, nil
		}
		return idx + 1, line, nil
	}
	if atEOF {
		line := bytes.TrimSpace(data)
		if len(line) == 0 {
			return len(data), nil, nil
		}
		return len(data), line, nil
	}
	return 0, nil, nil
}

// ParseStreamFixture parses a test fixture string into events.
func ParseStreamFixture(s string) ([]*Event, error) {
	dec := NewStreamDecoder(strings.NewReader(s))
	var out []*Event
	for {
		ev, err := dec.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}
