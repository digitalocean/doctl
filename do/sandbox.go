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
	"strings"

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
	Namespace string `json:"namespace"`
	APIHost   string `json:"api_host"`
	UUID      string `json:"uuid"`
	Key       string `json:"key"`
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
	// If DEBUG is specified, we need to open up stderr for that stream.  The stdout stream
	// will continue to work for returning structured results.
	if os.Getenv("DEBUG") != "" {
		cmd.Stderr = os.Stderr
	}
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
		APIHost: assignAPIHost(decoded.Namespace.APIHost, decoded.Namespace.Namespace),
		Auth:    decoded.Namespace.UUID + ":" + decoded.Namespace.Key,
	}
	return ans, nil
}

// Assign the correct API host based on the namespace name.
// Every serverless cluster has two domain names, one ending in '.io', the other in '.co'.
// By convention, the portal only returns the '.io' one but 'doctl sbx' must start using
// only the '.co' one (the '.io' one will eventually require mtls authentication).
// During a migration period, we can continue to support reconnection to "old" namespaces
// (not prefixed by "fn-") but should make sure that all "new" namespaces (prefixed by "fn-")
// switch their API host name from '.io' to '.co'.  Eventually, reconnection to old
// namespaces will fail and they will be removed.  We will need to ensure that users are
// using a doctl containing this code so they can obtain conforming namespaces.
func assignAPIHost(origAPIHost string, namespace string) string {
	if strings.HasPrefix(namespace, "fn-") {
		hostParts := strings.Split(origAPIHost, ".")
		sansSuffix := strings.Join(hostParts[:len(hostParts)-1], ".")
		return sansSuffix + ".co"
	}
	return origAPIHost
}
