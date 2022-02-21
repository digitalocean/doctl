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

// SandboxCredentials is the type returned by the ResolveToken and ResolveNamespace functions
// The values in it can be used to connect sandbox support to a specific namespace using the plugin.
type SandboxCredentials struct {
	Auth    string
	ApiHost string
}

// TokenRequest is the type of the request body for v2/function/namespaces/namespace when requesting
// the credentials for a JWT
type InputNamespace struct {
	Token string `json:"token"`
}
type TokenRequest struct {
	Namespace InputNamespace `json:"namespace"`
}

// TokenDecoded is the expected response field for v2/function/namespaces/namespace when requesting
// the credentials for a JWT
type OutputNamespace struct {
	ApiHost string `json:"api_host"`
	Uuid    string `json:"uuid"`
	Key     string `json:"key"`
}
type TokenDecoded struct {
	Namespace OutputNamespace `json:"namespace"`
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

// NewSandboxService returns a configure SandboxService.
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
	body := TokenRequest{Namespace: InputNamespace{Token: token}}
	req, err := n.client.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return SandboxCredentials{}, err
	}
	tokenDecoded := new(TokenDecoded)
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
	return SandboxCredentials{}, nil
}
