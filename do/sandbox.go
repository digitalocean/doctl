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
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/digitalocean/godo"
)

// SandboxCredentials models what is stored in credentials.json for use by the plugin and nim.
// It is also the type returned by the GetSandboxNamespace function.
type SandboxCredentials struct {
	APIHost     string                                  `json:"currentHost"`
	Namespace   string                                  `json:"currentNamespace"`
	Credentials map[string]map[string]SandboxCredential `json:"credentials"`
}

// SandboxCredential is the type of an individual entry in SandboxCredentials
type SandboxCredential struct {
	Auth string `json:"api_key"`
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

// ServerlessRuntime is the type of a runtime entry returned by the API host controller
// of the serverless cluster.
// Only relevant fields unmarshalled
type ServerlessRuntime struct {
	Default    bool   `json:"default"`
	Deprecated bool   `json:"deprecated"`
	Kind       string `json:"kind"`
}

// ServerlessHostInfo is the type of the host information return from the API host controller
// of the serverless cluster.
// Only relevant fields unmarshaled.
type ServerlessHostInfo struct {
	Runtimes map[string][]ServerlessRuntime `json:"runtimes"`
}

// FunctionInfo is the type of an individual function in the output
// of doctl sls fn list.  Only relevant fields are unmarshaled.
// Note: when we start replacing the sandbox plugin path with direct calls
// to backend controller operations, this will be replaced by declarations
// in the golang openwhisk client.
type FunctionInfo struct {
	Name        string       `json:"name"`
	Namespace   string       `json:"namespace"`
	Updated     int64        `json:"updated"`
	Version     string       `json:"version"`
	Annotations []Annotation `json:"annotations"`
}

// Annotation is a key/value type suitable for individual annotations
type Annotation struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// SandboxService is an interface for interacting with the sandbox plugin,
// with the namespaces service, and with the serverless cluster controller.
type SandboxService interface {
	Cmd(string, []string) (*exec.Cmd, error)
	Exec(*exec.Cmd) (SandboxOutput, error)
	Stream(*exec.Cmd) error
	GetSandboxNamespace(context.Context) (SandboxCredentials, error)
	WriteCredentials(SandboxCredentials) error
	GetHostInfo(string) (ServerlessHostInfo, error)
}

type sandboxService struct {
	sandboxJs string
	credsDir  string // note: this was misleadingly named sandboxDir previously
	node      string
	userAgent string
	client    *godo.Client
}

// CredentialsFile is the name of the file where the sandbox plugin stores OpenWhisk credentials.
const CredentialsFile = "credentials.json"

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
func NewSandboxService(sandboxJs string, credsDir string, node string, userAgent string, client *godo.Client) SandboxService {
	return &sandboxService{
		sandboxJs: sandboxJs,
		credsDir:  credsDir,
		node:      node,
		userAgent: userAgent,
		client:    client,
	}
}

// Cmd builds an *exec.Cmd for calling into the sandbox plugin.
func (n *sandboxService) Cmd(command string, args []string) (*exec.Cmd, error) {
	args = append([]string{n.sandboxJs, command}, args...)
	cmd := exec.Command(n.node, args...)
	cmd.Env = append(os.Environ(), "NIMBELLA_DIR="+n.credsDir, "NIM_USER_AGENT="+n.userAgent)
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
	req, err := n.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return SandboxCredentials{}, err
	}
	decoded := new(namespacesResponseBody)
	_, err = n.client.Do(ctx, req, decoded)
	if err != nil {
		return SandboxCredentials{}, err
	}
	host := assignAPIHost(decoded.Namespace.APIHost, decoded.Namespace.Namespace)
	credential := SandboxCredential{Auth: decoded.Namespace.UUID + ":" + decoded.Namespace.Key}
	namespace := decoded.Namespace.Namespace
	ans := SandboxCredentials{
		APIHost:     host,
		Namespace:   namespace,
		Credentials: map[string]map[string]SandboxCredential{host: {namespace: credential}},
	}
	return ans, nil
}

// GetHostInfo returns the HostInfo structure of the provided API host
func (n *sandboxService) GetHostInfo(APIHost string) (ServerlessHostInfo, error) {
	endpoint := APIHost + "/api/v1"
	resp, err := http.Get(endpoint)
	if err != nil {
		return ServerlessHostInfo{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	var result ServerlessHostInfo
	err = json.Unmarshal(body, &result)
	return result, err
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

// WriteCredentials writes a set of serverless credentials to the appropriate 'creds' directory
func (n *sandboxService) WriteCredentials(creds SandboxCredentials) error {
	// Create the directory into which the file will be written.
	err := os.MkdirAll(s.credsDir, 0700)
	if err != nil {
		return err
	}
	// Write the credentials
	credsPath := filepath.Join(n.credsDir, CredentialsFile)
	bytes, err := json.MarshalIndent(&creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(credsPath, bytes, 0600)
}
