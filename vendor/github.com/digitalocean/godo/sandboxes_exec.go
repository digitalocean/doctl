package godo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// SandboxExecPTY turns a PTY on for an exec. Zero fields use the defaults:
// 24 rows, 80 columns and xterm-256color.
type SandboxExecPTY struct {
	Rows int64  `json:"rows,omitempty"`
	Cols int64  `json:"cols,omitempty"`
	Term string `json:"term,omitempty"`
}

// SandboxExecStartRequest is the body of StartExec.
type SandboxExecStartRequest struct {
	// Argv is the executable and its arguments; Argv[0] is the executable.
	// Use {"bash", "-lc", "..."} for a shell. It can be empty only when PTY is
	// set, which starts the guest's default shell.
	Argv []string `json:"argv,omitempty"`
	// Workdir is the working directory.
	Workdir string `json:"workdir,omitempty"`
	// EnvVars are environment variables for the process.
	EnvVars map[string]string `json:"env_vars,omitempty"`
	// TimeoutSeconds bounds the stream. Zero means the service default. The API
	// encodes it as a string.
	TimeoutSeconds int64 `json:"timeout_seconds,string,omitempty"`
	// KeepStdinOpen keeps stdin open for WriteExecStdin calls. The stream then
	// starts with a Started event that carries the exec ID.
	KeepStdinOpen bool `json:"keep_stdin_open,omitempty"`
	// PTY allocates a terminal when set.
	PTY *SandboxExecPTY `json:"pty,omitempty"`
	// Stdin is a first stdin chunk sent with the start request. It is
	// base64-encoded on the wire.
	Stdin []byte `json:"stdin,omitempty"`
}

// SandboxExecStdinRequest is the body of WriteExecStdin.
type SandboxExecStdinRequest struct {
	// Data is written to stdin. It is base64-encoded on the wire.
	Data []byte `json:"data,omitempty"`
	// Offset is the number of bytes the guest has already accepted: zero for
	// the first write, then the offset the previous write returned. The API
	// encodes it as a string.
	Offset int64 `json:"offset,string"`
	// EOF closes stdin after writing (on a PTY it sends VEOF).
	EOF bool `json:"eof,omitempty"`
}

// SandboxExecStarted is the first event of an exec started with KeepStdinOpen
// or PTY. A plain exec does not send it.
type SandboxExecStarted struct {
	ExecID string `json:"exec_id"`
}

// SandboxExecExit is the last event of an exec's stream. A missing exit code
// is zero.
type SandboxExecExit struct {
	ExitCode   int32  `json:"exit_code"`
	DurationMS int64  `json:"duration_ms,string,omitempty"`
	Status     string `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
}

// SandboxExecEvent is one event on an exec's stream. Exactly one field is set.
// Stdout and Stderr are raw bytes (base64 on the wire).
type SandboxExecEvent struct {
	Started   *SandboxExecStarted `json:"started,omitempty"`
	Stdout    []byte              `json:"stdout,omitempty"`
	Stderr    []byte              `json:"stderr,omitempty"`
	Exit      *SandboxExecExit    `json:"exit,omitempty"`
	Keepalive *struct{}           `json:"keepalive,omitempty"`
}

// SandboxExecStream reads the events of an exec. Close it when done.
type SandboxExecStream struct {
	body io.ReadCloser
	dec  *json.Decoder
}

// Next returns the next event, or io.EOF when the stream ends. A stream error
// is returned as *SandboxCommandStreamError.
func (s *SandboxExecStream) Next() (*SandboxExecEvent, error) {
	return nextSandboxStreamEvent[SandboxExecEvent](s.dec)
}

// Close closes the stream. The process may keep running.
func (s *SandboxExecStream) Close() error { return s.body.Close() }

func sandboxExecPath(sandboxID string) string {
	return fmt.Sprintf("%s/%s/exec", sandboxesBasePath, sandboxID)
}

func sandboxExecActionPath(sandboxID, execID, action string) string {
	return fmt.Sprintf("%s/%s/%s", sandboxExecPath(sandboxID), execID, action)
}

// StartExec runs a process and streams its events on the same request, so no
// output is missed. The stream stays open until the process exits or ctx ends.
// A failing executable is not an error: it is an Exit event with its exit code.
// A request the API rejects returns an *ErrorResponse before streaming.
func (s *SandboxesServiceOp) StartExec(ctx context.Context, sandboxID string, start *SandboxExecStartRequest) (*SandboxExecStream, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	if start == nil || (len(start.Argv) == 0 && start.PTY == nil) {
		return nil, nil, errors.New("sandboxes: argv is required unless pty is set")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, sandboxExecPath(sandboxID), start)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.DoStream(ctx, req)
	if err != nil {
		return nil, resp, err
	}
	return &SandboxExecStream{body: resp.Body, dec: json.NewDecoder(resp.Body)}, resp, nil
}

// WriteExecStdin writes to the stdin of an exec started with KeepStdinOpen or
// PTY, and returns the number of bytes the guest has accepted after this
// write. Pass that as the Offset of the next write.
func (s *SandboxesServiceOp) WriteExecStdin(ctx context.Context, sandboxID, execID string, in *SandboxExecStdinRequest) (int64, *Response, error) {
	if sandboxID == "" {
		return 0, nil, errors.New("sandboxes: sandbox id is required")
	}
	if execID == "" {
		return 0, nil, errors.New("sandboxes: exec id is required")
	}
	if in == nil {
		return 0, nil, errors.New("sandboxes: stdin request is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, sandboxExecActionPath(sandboxID, execID, "stdin"), in)
	if err != nil {
		return 0, nil, err
	}
	root := new(struct {
		Offset int64 `json:"offset,string,omitempty"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return 0, resp, err
	}
	return root.Offset, resp, nil
}

// ResizeExec resizes the terminal of an exec started with PTY. Rows and cols
// must be between 1 and 1000.
func (s *SandboxesServiceOp) ResizeExec(ctx context.Context, sandboxID, execID string, rows, cols int64) (*Response, error) {
	if sandboxID == "" {
		return nil, errors.New("sandboxes: sandbox id is required")
	}
	if execID == "" {
		return nil, errors.New("sandboxes: exec id is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, sandboxExecActionPath(sandboxID, execID, "resize"),
		map[string]int64{"rows": rows, "cols": cols})
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
