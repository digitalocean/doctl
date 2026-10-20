/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package do

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/digitalocean/godo"
)

// sandboxExecPath is the harness-api route for one-shot / streaming exec.
// Kept local so streaming does not need a godo bump; matches godo's
// hostedAgentSessionSandboxExecPath.
const sandboxExecPath = "/v2/agents/sessions/%s/sandbox/exec"

// SandboxExecEvent is one frame from StreamExecInSandbox. At most one of
// Stdout, Stderr, ExitCode, or ErrMessage is set.
type SandboxExecEvent struct {
	Stdout     []byte
	Stderr     []byte
	ExitCode   *int
	ErrMessage string
}

type execSSEChunkData struct {
	Data string `json:"data"`
}

type execSSEExitData struct {
	ExitCode   int   `json:"exit_code"`
	DurationMS int64 `json:"duration_ms"`
}

type execSSEErrorData struct {
	Message string `json:"message"`
}

func (s *hostedAgentsService) StreamExecInSandbox(ctx context.Context, sessionID string, body *godo.HostedAgentSandboxExecRequest, emit func(SandboxExecEvent) error) error {
	if emit == nil {
		return errors.New("hosted agents: stream exec requires an emit callback")
	}
	if sessionID == "" {
		return errors.New("hosted agents: session id is required")
	}
	if body == nil || len(body.Argv) == 0 {
		return errors.New("hosted agents: argv is required")
	}

	path := fmt.Sprintf(sandboxExecPath, sessionID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := s.client.DoStream(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		// Pre-SSE harness (or any proxy that strips Accept) still returns the
		// buffered JSON document. Replay it as the same event sequence.
		return emitBufferedExecAsStream(resp.Body, emit)
	}
	return readExecSSE(resp.Body, emit)
}

func emitBufferedExecAsStream(r io.Reader, emit func(SandboxExecEvent) error) error {
	var root godo.HostedAgentSandboxExecResponse
	if err := json.NewDecoder(r).Decode(&root); err != nil {
		return fmt.Errorf("hosted agents: decode buffered exec response: %w", err)
	}
	if root.Stdout != "" {
		if err := emit(SandboxExecEvent{Stdout: []byte(root.Stdout)}); err != nil {
			return err
		}
	}
	if root.Stderr != "" {
		if err := emit(SandboxExecEvent{Stderr: []byte(root.Stderr)}); err != nil {
			return err
		}
	}
	code := root.ExitCode
	return emit(SandboxExecEvent{ExitCode: &code})
}

func readExecSSE(r io.Reader, emit func(SandboxExecEvent) error) error {
	reader := godo.NewSSEReader(r)
	sawExit := false
	for {
		ev, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		switch ev.Event {
		case "stdout", "stderr":
			var chunk execSSEChunkData
			if err := json.Unmarshal(ev.Data, &chunk); err != nil {
				return fmt.Errorf("hosted agents: decode exec %s event: %w", ev.Event, err)
			}
			raw, err := base64.StdEncoding.DecodeString(chunk.Data)
			if err != nil {
				return fmt.Errorf("hosted agents: decode exec %s payload: %w", ev.Event, err)
			}
			frame := SandboxExecEvent{}
			if ev.Event == "stdout" {
				frame.Stdout = raw
			} else {
				frame.Stderr = raw
			}
			if err := emit(frame); err != nil {
				return err
			}
		case "exit":
			var exit execSSEExitData
			if err := json.Unmarshal(ev.Data, &exit); err != nil {
				return fmt.Errorf("hosted agents: decode exec exit event: %w", err)
			}
			code := exit.ExitCode
			sawExit = true
			if err := emit(SandboxExecEvent{ExitCode: &code}); err != nil {
				return err
			}
		case "error":
			var er execSSEErrorData
			if err := json.Unmarshal(ev.Data, &er); err != nil {
				return fmt.Errorf("hosted agents: decode exec error event: %w", err)
			}
			msg := strings.TrimSpace(er.Message)
			if msg == "" {
				msg = "exec failed"
			}
			return errors.New(msg)
		default:
			// Ignore unknown / comment-driven keepalives (comments never land here).
		}
	}
	if !sawExit {
		return errors.New("hosted agents: exec stream ended without exit")
	}
	return nil
}
