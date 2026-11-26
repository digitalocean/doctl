package godo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// SandboxCommand is a command running in a sandbox.
type SandboxCommand struct {
	PID int64  `json:"pid"`
	Cmd string `json:"cmd"`
	Cwd string `json:"cwd,omitempty"`
}

// SandboxCommandStartRequest is the body of StartCommand.
type SandboxCommandStartRequest struct {
	// Cmd is the executable, not a shell string. Use "bash" with
	// Args {"-lc", "..."} for a shell.
	Cmd string `json:"cmd"`
	// Args are passed to the executable.
	Args []string `json:"args,omitempty"`
	// Cwd is the working directory.
	Cwd string `json:"cwd,omitempty"`
	// EnvVars are environment variables for the command.
	EnvVars map[string]string `json:"env_vars,omitempty"`
	// Stdin leaves stdin open for later WriteCommandStdin calls.
	Stdin bool `json:"stdin,omitempty"`
	// TimeoutSeconds kills the command after this long. Zero means no timeout.
	// The API encodes it as a string.
	TimeoutSeconds int64 `json:"timeout_seconds,omitempty,string"`
}

// SandboxCommandSignal is a signal that SignalCommand can send.
type SandboxCommandSignal string

const (
	// SandboxCommandSignalTerm is SIGTERM, the default.
	SandboxCommandSignalTerm SandboxCommandSignal = "SANDBOX_COMMAND_SIGNAL_SIGTERM"
	// SandboxCommandSignalKill is SIGKILL.
	SandboxCommandSignalKill SandboxCommandSignal = "SANDBOX_COMMAND_SIGNAL_SIGKILL"
)

// SandboxCommandStdinRequest is the body of WriteCommandStdin.
type SandboxCommandStdinRequest struct {
	// Data is written to stdin. It is base64-encoded on the wire.
	Data []byte `json:"data,omitempty"`
	// EOF closes stdin after writing.
	EOF bool `json:"eof,omitempty"`
}

// SandboxCommandStarted is the first event of a command's stream.
type SandboxCommandStarted struct {
	PID int64 `json:"pid"`
}

// SandboxCommandExit is the last event of a command's stream.
type SandboxCommandExit struct {
	ExitCode int32  `json:"exit_code"`
	Status   string `json:"status,omitempty"`
	Error    string `json:"error,omitempty"`
}

// SandboxCommandEvent is one event on a command's stream. Exactly one field is
// set. Stdout and Stderr are raw bytes (base64 on the wire).
type SandboxCommandEvent struct {
	Started   *SandboxCommandStarted `json:"started,omitempty"`
	Stdout    []byte                 `json:"stdout,omitempty"`
	Stderr    []byte                 `json:"stderr,omitempty"`
	Exit      *SandboxCommandExit    `json:"exit,omitempty"`
	Keepalive *struct{}              `json:"keepalive,omitempty"`
}

// SandboxCommandStreamError is a transport error the gateway sent in the
// stream. The command and exec streams both return it.
type SandboxCommandStreamError struct {
	GRPCCode   int32  `json:"grpc_code"`
	HTTPCode   int32  `json:"http_code"`
	Message    string `json:"message"`
	HTTPStatus string `json:"http_status"`
}

func (e *SandboxCommandStreamError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("sandbox command stream failed (HTTP %d)", e.HTTPCode)
	}
	return e.Message
}

// SandboxCommandStream reads the events of a running command. Close it when done.
type SandboxCommandStream struct {
	body io.ReadCloser
	dec  *json.Decoder
}

// Next returns the next event, or io.EOF when the stream ends. It skips
// envelopes that carry neither a result nor an error. A stream error is
// returned as *SandboxCommandStreamError.
func (s *SandboxCommandStream) Next() (*SandboxCommandEvent, error) {
	return nextSandboxStreamEvent[SandboxCommandEvent](s.dec)
}

// nextSandboxStreamEvent decodes the next {"result": ...} or {"error": ...}
// envelope of a grpc-gateway stream. It skips envelopes that carry neither,
// and returns io.EOF when the stream ends.
func nextSandboxStreamEvent[T any](dec *json.Decoder) (*T, error) {
	for {
		var envelope struct {
			Result *T                         `json:"result"`
			Error  *SandboxCommandStreamError `json:"error"`
		}
		if err := dec.Decode(&envelope); err != nil {
			return nil, err
		}
		if envelope.Error != nil {
			return nil, envelope.Error
		}
		if envelope.Result != nil {
			return envelope.Result, nil
		}
	}
}

// Close closes the stream.
func (s *SandboxCommandStream) Close() error { return s.body.Close() }

func sandboxCommandsPath(sandboxID string) string {
	return fmt.Sprintf("%s/%s/commands", sandboxesBasePath, sandboxID)
}

func sandboxCommandPath(sandboxID string, pid int64, action string) string {
	return fmt.Sprintf("%s/%d/%s", sandboxCommandsPath(sandboxID), pid, action)
}

// ListCommands lists the commands running in a sandbox.
func (s *SandboxesServiceOp) ListCommands(ctx context.Context, sandboxID string) ([]SandboxCommand, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, sandboxCommandsPath(sandboxID), nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(struct {
		Commands []SandboxCommand `json:"commands"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.Commands, resp, nil
}

// StartCommand starts a command and returns its pid. The process keeps running
// after the call returns; read its output with StreamCommand.
func (s *SandboxesServiceOp) StartCommand(ctx context.Context, sandboxID string, start *SandboxCommandStartRequest) (int64, *Response, error) {
	if sandboxID == "" {
		return 0, nil, errors.New("sandboxes: sandbox id is required")
	}
	if start == nil || start.Cmd == "" {
		return 0, nil, errors.New("sandboxes: command is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, sandboxCommandsPath(sandboxID), start)
	if err != nil {
		return 0, nil, err
	}
	root := new(struct {
		PID *int64 `json:"pid"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return 0, resp, err
	}
	if root.PID == nil {
		return 0, resp, errors.New("sandboxes: the response did not include the command's pid")
	}
	return *root.PID, resp, nil
}

// StreamCommand attaches to a running command and streams its events. The
// stream stays open until the command exits or ctx ends. An unknown or exited
// pid returns a 404 *ErrorResponse.
func (s *SandboxesServiceOp) StreamCommand(ctx context.Context, sandboxID string, pid int64) (*SandboxCommandStream, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, sandboxCommandPath(sandboxID, pid, "stream"), nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.DoStream(ctx, req)
	if err != nil {
		return nil, resp, err
	}
	return &SandboxCommandStream{body: resp.Body, dec: json.NewDecoder(resp.Body)}, resp, nil
}

// SignalCommand sends a signal to a command. An empty signal sends SIGTERM.
func (s *SandboxesServiceOp) SignalCommand(ctx context.Context, sandboxID string, pid int64, signal SandboxCommandSignal) (*Response, error) {
	if sandboxID == "" {
		return nil, errors.New("sandboxes: sandbox id is required")
	}
	if signal == "" {
		signal = SandboxCommandSignalTerm
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, sandboxCommandPath(sandboxID, pid, "signal"),
		map[string]SandboxCommandSignal{"signal": signal})
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// WriteCommandStdin writes to the stdin of a command started with Stdin set.
func (s *SandboxesServiceOp) WriteCommandStdin(ctx context.Context, sandboxID string, pid int64, in *SandboxCommandStdinRequest) (*Response, error) {
	if sandboxID == "" {
		return nil, errors.New("sandboxes: sandbox id is required")
	}
	if in == nil {
		return nil, errors.New("sandboxes: stdin request is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, sandboxCommandPath(sandboxID, pid, "stdin"), in)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
