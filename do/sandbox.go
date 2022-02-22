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

// SandboxCredentials is the type returned by the ResolveToken and ResolveNamespace functions
// The values in it can be used to connect sandbox support to a specific namespace using the plugin.
type SandboxCredentials struct {
	Auth    string
	ApiHost string
}

// The type of the "namespace" member of POST input for API calls.
// Only one of the fields is typically specified
type inputNamespace struct {
	Token     string `json:"token,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// The type of the "namespace" member of the response to API calls.  Only some
// fields are relevant to each call.
type outputNamespace struct {
	ApiHost   string `json:"api_host"`
	Uuid      string `json:"uuid"`
	Key       string `json:"key"`
	Token     string `json:"token"`
	Label     string `json:"label"`
	Namespace string `json:"namespace"`
}

// postBody is the type of the request body for v2/function/namespaces/... calls that use
// the POST verb.  Only the relevant parts of the contained inputNamespace need be specified
type postBody struct {
	Namespace inputNamespace `json:"namespace"`
}

// responseBody is the type of the response body from API calls other than "list namespaces".
type responseBody struct {
	Namespace outputNamespace `json:"namespace"`
}

// namespaceList is the type of the response body from "list namespaces"
type namespaceList struct {
	Namespaces []outputNamespace `json:"namespaces"`
}

// SandboxService is an interface for interacting with the sandbox plugin
// and with the namespaces service.
type SandboxService interface {
	Cmd(string, []string) (*exec.Cmd, error)
	Exec(*exec.Cmd) (SandboxOutput, error)
	Stream(*exec.Cmd) error
	ResolveToken(context.Context, string) (SandboxCredentials, error)
	ResolveNamespace(context.Context, string) (SandboxCredentials, error)
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
	// Result is sound JSON but if it has an Error field the rest is not trustworthy
	if len(result.Error) > 0 {
		return SandboxOutput{}, errors.New(result.Error)
	}
	// Result is both sound and error free
	return result, nil
}

// Stream is like Exec but assumes that output will not be captured and can be streamed.
func (n *sandboxService) Stream(cmd *exec.Cmd) error {

	return cmd.Run()
}

// ResolveToken resolves a JWT issued by the UI into a set of credentials for the sandbox
// Note: the use of JWTs may eventually go away in favor of going straight from namespace
// name to the actual tokens.
func (n *sandboxService) ResolveToken(ctx context.Context, token string) (SandboxCredentials, error) {
	path := "v2/function/namespaces/namespace"
	body := postBody{Namespace: inputNamespace{Token: token}}
	req, err := n.client.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return SandboxCredentials{}, err
	}
	tokenDecoded := new(responseBody)
	_, err = n.client.Do(ctx, req, tokenDecoded)
	if err != nil {
		return SandboxCredentials{}, err
	}
	ans := SandboxCredentials{
		ApiHost: tokenDecoded.Namespace.ApiHost,
		Auth:    tokenDecoded.Namespace.Uuid + ":" + tokenDecoded.Namespace.Key,
	}
	return ans, nil
}

// ResolveNamespace resolves a namespace name into a set of credentials for the sandbox.
// If "" is given as the namespace name, the available namespaces are retrieved and the
// function attempts to identify one of them as the sandbox namespace.
// Note: at present, the "" option only works when the customer has exactly one namespace
// whose 'label' field contains the substring 'sandbox'.   This is subject to change.
// Note: at present, two remote calls are needed to go from a namespace name to the needed
// credentials.  First, a JWT token is remotely generated, then ResolveToken call is used
// to resolve it.  This will be streamlined in the future.
func (n *sandboxService) ResolveNamespace(ctx context.Context, namespace string) (SandboxCredentials, error) {
	var err error
	if namespace == "" {
		namespace, err = findSandboxNamespace(n.client, ctx)
		if err != nil {
			return SandboxCredentials{}, err
		}
	}
	token, err := getTokenForNamespace(n.client, ctx, namespace)
	if err != nil {
		return SandboxCredentials{}, err
	}
	return n.ResolveToken(ctx, token)
}

// getTokenForNamespace is a subroutine of ResolveNamespace wrapping the call to
// obtain a JWT.  This step should eventually be eliminated in favor of an API
// that goes directly from namespace to credentials.
func getTokenForNamespace(client *godo.Client, ctx context.Context, namespace string) (string, error) {
	path := "v2/function/namespaces/login_token"
	body := postBody{Namespace: inputNamespace{Namespace: namespace}}
	req, err := client.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return "", err
	}
	tokenResponse := new(responseBody)
	_, err = client.Do(ctx, req, tokenResponse)
	if err != nil {
		return "", err
	}
	return tokenResponse.Namespace.Token, nil
}

// findSandboxNamespace is a subroutine of ResolveNamespace implementing the search for
// a sandbox namespace.
func findSandboxNamespace(client *godo.Client, ctx context.Context) (string, error) {
	path := "v2/function/namespaces"
	req, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	namespaces := new(namespaceList)
	_, err = client.Do(ctx, req, namespaces)
	if err != nil {
		return "", err
	}
	sandboxes := []string{}
	for _, ns := range namespaces.Namespaces {
		if strings.Contains(ns.Label, "sandbox") {
			sandboxes = append(sandboxes, ns.Namespace)
		}
	}
	if len(sandboxes) == 1 {
		return sandboxes[0], nil
	}
	return "", errors.New("could not find a sandbox namespace in the cloud for this account")
}
