/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"

	"github.com/digitalocean/godo"
)

// SandboxCredentials is the type returned by the GetSandboxNamespace function.
// The values in it can be used to connect sandbox support to a specific namespace using the plugin.
type SandboxCredentials struct {
	Auth    string
	APIHost string
}

// The type of the "namespace" member of the response to /api/v2/functions/sandbox
// Only relevant fields unmarshalled
type outputNamespace struct {
	APIHost string `json:"api_host"`
	UUID    string `json:"uuid"`
	Key     string `json:"key"`
}

// namespacesResponseBody is the type of the response body for /api/v2/functions/sandbox
type namespacesResponseBody struct {
	Namespace outputNamespace `json:"namespace"`
}

// SandboxService is an interface for interacting with the sandbox plugin
// and with the namespaces service.
type SandboxService interface {
	Cmd(string, []string) (*exec.Cmd, error)
	Exec(*exec.Cmd) (SandboxOutput, error)
	Stream(*exec.Cmd) error
	GetSandboxNamespace(context.Context) (SandboxCredentials, error)
}

type sandboxService struct {
	sandboxJs  string
	sandboxDir string
	node       string
	client     *godo.Client
}

var _ SandboxService = &sandboxService{}

// SandboxOutput contains the output returned from calls to the sandbox plugin.
type SandboxOutput struct {
	Table     []map[string]interface{} `json:"table,omitempty"`
	Captured  []string                 `json:"captured,omitempty"`
	Formatted []string                 `json:"formatted,omitempty"`
	Entity    interface{}              `json:"entity,omitempty"`
	Error     string                   `json:"error,omitempty"`
}

// NewSandboxService returns a configured SandboxService.
func NewSandboxService(sandboxJs string, sandboxDir string, node string, client *godo.Client) SandboxService {
	return &sandboxService{
		sandboxJs:  sandboxJs,
		sandboxDir: sandboxDir,
		node:       node,
		client:     client,
	}
}

// Cmd builds an *exec.Cmd for calling into the sandbox plugin.
func (n *sandboxService) Cmd(command string, args []string) (*exec.Cmd, error) {
	args = append([]string{n.sandboxJs, command}, args...)
	cmd := exec.Command(n.node, args...)
	cmd.Env = append(os.Environ(), "NIMBELLA_DIR="+n.sandboxDir)

	return cmd, nil
}

// Exec executes an *exec.Cmd and captures its output in a SandboxOutput.
func (n *sandboxService) Exec(cmd *exec.Cmd) (SandboxOutput, error) {
	output, err := cmd.Output()
	if err != nil {
		// Ignore "errors" that are just non-zero exit.  The
		// sandbox uses this as a secondary indicator but the output
		// is still trustworthy (and includes error information inline)
		if _, ok := err.(*exec.ExitError); !ok {
			// Real error of some sort
			return SandboxOutput{}, err
		}
	}
	var result SandboxOutput
	err = json.Unmarshal(output, &result)
	if err != nil {
		return SandboxOutput{}, err
	}
	// Result is sound JSON but also has an error field, meaning that something did
	// go wrong.  In this case we return the actual output but also the distinguished
	// error return.  Most callers will process only the error, which is fine.  Sometimes,
	// however, there is other information that can be useful as part of the error report.
	if len(result.Error) > 0 {
		return result, errors.New(result.Error)
	}
	// Result is both sound and error free
	return result, nil
}

// Stream is like Exec but assumes that output will not be captured and can be streamed.
func (n *sandboxService) Stream(cmd *exec.Cmd) error {

	return cmd.Run()
}

// GetSandboxNamespace returns the credentials of the one sandbox namespace assigned to
// the invoking doctl context.
func (n *sandboxService) GetSandboxNamespace(ctx context.Context) (SandboxCredentials, error) {
	path := "v2/functions/sandbox"
	req, err := n.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return SandboxCredentials{}, err
	}
	decoded := new(namespacesResponseBody)
	_, err = n.client.Do(ctx, req, decoded)
	if err != nil {
		return SandboxCredentials{}, err
	}
	ans := SandboxCredentials{
		APIHost: decoded.Namespace.APIHost,
		Auth:    decoded.Namespace.UUID + ":" + decoded.Namespace.Key,
	}
	return ans, nil
}
