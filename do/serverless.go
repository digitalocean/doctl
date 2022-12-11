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
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/apache/openwhisk-client-go/whisk"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/pkg/extract"
	"github.com/digitalocean/godo"
	"github.com/pkg/browser"
	"gopkg.in/yaml.v3"
)

// ServerlessCredentials models what is stored in credentials.json for use by the plugin and nim.
// It is also the type returned by the GetServerlessNamespace function.
type ServerlessCredentials struct {
	APIHost     string                                     `json:"currentHost"`
	Namespace   string                                     `json:"currentNamespace"`
	Label       string                                     `json:"label"`
	Credentials map[string]map[string]ServerlessCredential `json:"credentials"`
}

// ServerlessCredential is the type of an individual entry in ServerlessCredentials
type ServerlessCredential struct {
	Auth string `json:"api_key"`
}

// OutputNamespace is the type of the "namespace" member of the response to /api/v2/functions/sandbox
// and /api/v2/functions/namespaces APIs.  Only relevant fields unmarshalled
type OutputNamespace struct {
	Namespace string `json:"namespace"`
	APIHost   string `json:"api_host"`
	UUID      string `json:"uuid"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	Region    string `json:"Region"`
}

// FunctionParameter is the type of a parameter in the response body of action.get.  We do our
// own JSON unmarshaling of these because the go OpenWhisk client doesn't include the "init" and
// "encryption" members, of which at least "init" is needed.
type FunctionParameter struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Init       bool   `json:"init"`
	Encryption string `json:"encryption"`
}

// FunctionParameterReparse is a partial remapping of whisk.Action so that the parameters
// are declared as FunctionParameter rather than whisk.KeyValue.
type FunctionParameterReparse struct {
	Parameters []FunctionParameter `json:"parameters"`
}

// NamespaceResponse is the type of the response body for /api/v2/functions/sandbox (POST) and
// /api/v2/functions/namespaces/<nsName> (GET)
type NamespaceResponse struct {
	Namespace OutputNamespace `json:"namespace"`
}

// NamespaceListResponse is the type of the response body for /api/v2/functions/namespaces (GET)
type NamespaceListResponse struct {
	Namespaces []OutputNamespace `json:"namespaces"`
}

// newNamespaceRequest is the type of the POST body for requesting a new namespace
type newNamespaceRequest struct {
	Namespace inputNamespace `json:"namespace"`
}

// inputNamespace is the reduced representation of a namespace used when requesting a new one
type inputNamespace struct {
	Label  string `json:"label"`
	Region string `json:"Region"`
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

// ServerlessProject ...
type ServerlessProject struct {
	ProjectPath string   `json:"project_path"`
	ConfigPath  string   `json:"config"`
	Packages    string   `json:"packages"`
	Env         string   `json:"env"`
	Strays      []string `json:"strays"`
}

// ServerlessSpec describes a project.yml spec
// reference: https://docs.nimbella.com/configuration/
type ServerlessSpec struct {
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Environment map[string]interface{} `json:"environment,omitempty"`
	Packages    []*ServerlessPackage   `json:"packages,omitempty"`
}

// ServerlessPackage ...
type ServerlessPackage struct {
	Name        string                 `json:"name,omitempty"`
	Shared      bool                   `json:"shared,omitempty"`
	Environment map[string]interface{} `json:"environment,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
	Functions   []*ServerlessFunction  `json:"functions,omitempty"`
}

// ServerlessFunction ...
type ServerlessFunction struct {
	Name    string `json:"name,omitempty"`
	Binary  bool   `json:"binary,omitempty"`
	Main    string `json:"main,omitempty"`
	Runtime string `json:"runtime,omitempty"`
	// `web` can be either true or "raw". We use interface{} to support both types. If we start consuming the value we
	// should probably define a custom type with proper validation.
	Web         interface{}            `json:"web,omitempty"`
	WebSecure   interface{}            `json:"webSecure,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Environment map[string]interface{} `json:"environment,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
	Limits      map[string]int         `json:"limits,omitempty"`
}

// ProjectMetadata describes the nim project:get-metadata output structure.
type ProjectMetadata struct {
	ServerlessSpec
	UnresolvedVariables []string `json:"unresolvedVariables,omitempty"`
}

// ServerlessTriggerListResponse is the form returned by the list triggers API
type ServerlessTriggerListResponse struct {
	Triggers []ServerlessTrigger `json:"Triggers,omitempty"`
}

// ServerlessTriggerGetResponse is the form returned by the get trigger API
type ServerlessTriggerGetResponse struct {
	Trigger ServerlessTrigger `json:"Trigger,omitempty"`
}

type UpdateTriggerRequest struct {
	IsEnabled        bool                     `json:"is_enabled"`
	ScheduledDetails *TriggerScheduledDetails `json:"scheduled_details,omitempty"`
}

// ServerlessTrigger is the form used in list and get responses by the triggers API
type ServerlessTrigger struct {
	Namespace        string                   `json:"namespace,omitempty"`
	Function         string                   `json:"function,omitempty"`
	Type             string                   `json:"type,omitempty"`
	Name             string                   `json:"name,omitempty"`
	IsEnabled        bool                     `json:"is_enabled"`
	CreatedAt        time.Time                `json:"created_at,omitempty"`
	UpdatedAt        time.Time                `json:"updated_at,omitempty"`
	ScheduledDetails *TriggerScheduledDetails `json:"scheduled_details,omitempty"`
	ScheduledRuns    *TriggerScheduledRuns    `json:"scheduled_runs,omitempty"`
}

type TriggerScheduledDetails struct {
	Cron string                 `json:"cron,omitempty"`
	Body map[string]interface{} `json:"body,omitempty"`
}

type TriggerScheduledRuns struct {
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
}

// ServerlessService is an interface for interacting with the sandbox plugin,
// with the namespaces service, and with the serverless cluster controller.
type ServerlessService interface {
	Cmd(string, []string) (*exec.Cmd, error)
	Exec(*exec.Cmd) (ServerlessOutput, error)
	Stream(*exec.Cmd) error
	GetServerlessNamespace(context.Context) (ServerlessCredentials, error)
	ListNamespaces(context.Context) (NamespaceListResponse, error)
	GetNamespace(context.Context, string) (ServerlessCredentials, error)
	GetNamespaceFromCluster(string, string) (string, error)
	CreateNamespace(context.Context, string, string) (ServerlessCredentials, error)
	DeleteNamespace(context.Context, string) error
	CleanNamespace() error
	ListTriggers(context.Context, string) ([]ServerlessTrigger, error)
	GetTrigger(context.Context, string) (ServerlessTrigger, error)
	UpdateTrigger(context.Context, string, *UpdateTriggerRequest) (ServerlessTrigger, error)
	DeleteTrigger(context.Context, string) error
	WriteCredentials(ServerlessCredentials) error
	ReadCredentials() (ServerlessCredentials, error)
	GetHostInfo(string) (ServerlessHostInfo, error)
	CheckServerlessStatus() error
	InstallServerless(string, bool) error
	ListPackages() ([]whisk.Package, error)
	DeletePackage(string, bool) error
	GetFunction(string, bool) (whisk.Action, []FunctionParameter, error)
	ListFunctions(string, int, int) ([]whisk.Action, error)
	DeleteFunction(string, bool) error
	InvokeFunction(string, interface{}, bool, bool) (interface{}, error)
	InvokeFunctionViaWeb(string, interface{}) error
	ListActivations(whisk.ActivationListOptions) ([]whisk.Activation, error)
	GetActivationCount(whisk.ActivationCountOptions) (whisk.ActivationCount, error)
	GetActivation(string) (whisk.Activation, error)
	GetActivationLogs(string) (whisk.Activation, error)
	GetActivationResult(string) (whisk.Response, error)
	GetConnectedAPIHost() (string, error)
	ReadProject(*ServerlessProject, []string) (ServerlessOutput, error)
	WriteProject(ServerlessProject) (string, error)
	SetEffectiveCredentials(auth string, apihost string)
	CredentialsPath() string
}

type serverlessService struct {
	serverlessJs  string
	serverlessDir string
	credsDir      string
	node          string
	userAgent     string
	accessToken   string
	client        *godo.Client
	owClient      *whisk.Client
	owConfig      *whisk.Config
}

const (
	// Minimum required version of the functions deployer plugin code.
	minServerlessVersion = "5.0.18"

	// The version of nodejs to download alongsize the plugin download.
	nodeVersion = "v16.13.0"

	// credsDir is the directory under the sandbox where all credentials are stored.
	// It in turn has a subdirectory for each access token employed (formed as a prefix of the token).
	credsDir = "creds"

	// CredentialsFile is the name of the file where the sandbox plugin stores OpenWhisk credentials.
	CredentialsFile = "credentials.json"
)

const (
	/*
		The following are forbidden configurations for a serverless project.
		Validation ensures these are the configurations are not set in the project.yml
		Some of these configs can exist at multiple levels (i.e. Namespace, package, and action)
	*/

	// ForbiddenConfigShared ...
	ForbiddenConfigShared = "shared"
	// ForbiddenConfigWebSecure ...
	ForbiddenConfigWebSecure = "webSecure"
	// ForbiddenConfigSequence ...
	ForbiddenConfigSequence = "sequence"
	// ForbiddenConfigProvideAPIKeyAnnotation ...
	ForbiddenConfigProvideAPIKeyAnnotation = "provideAPIKeyAnnotation"
	// ForbiddenConfigRequireWhiskAuthAnnotation ...
	ForbiddenConfigRequireWhiskAuthAnnotation = "provideWhiskAuthAnnotation"

	/*
		These are values for forbidden annotations. Not all annotations are forbidden
	*/

	// ForbiddenAnnotationProvideAPIKey ...
	ForbiddenAnnotationProvideAPIKey = "provide-api-key"
	// ForbiddenAnnotationRequireWhiskAuth ...
	ForbiddenAnnotationRequireWhiskAuth = "require-whisk-auth"
)

var _ ServerlessService = &serverlessService{}

var (
	// ErrServerlessNotInstalled is the error returned to users when the sandbox is not installed.
	ErrServerlessNotInstalled = errors.New("serverless support is not installed (use `doctl serverless install`)")

	// ErrServerlessNeedsUpgrade is the error returned to users when the sandbox is at too low a version
	ErrServerlessNeedsUpgrade = errors.New("serverless support needs to be upgraded (use `doctl serverless upgrade`)")

	// ErrServerlessNotConnected is the error returned to users when the sandbox is not connected to a namespace
	ErrServerlessNotConnected = errors.New("serverless support is installed but not connected to a functions namespace (use `doctl serverless connect`)")
)

// ServerlessOutput contains the output returned from calls to the sandbox plugin.
type ServerlessOutput struct {
	Table     []map[string]interface{} `json:"table,omitempty"`
	Captured  []string                 `json:"captured,omitempty"`
	Formatted []string                 `json:"formatted,omitempty"`
	Entity    interface{}              `json:"entity,omitempty"`
	Error     string                   `json:"error,omitempty"`
}

// NewServerlessService returns a configured ServerlessService.
func NewServerlessService(client *godo.Client, usualServerlessDir string, accessToken string) ServerlessService {
	nodeBin := "node"
	if runtime.GOOS == "windows" {
		nodeBin = "node.exe"
	}
	// The following is needed to support snap installation.  For snap, the installation directory
	// is relocated to a snap-managed area.  That area is not user-writable, so, the credsDir location
	// is always computed relative to the normal installation area (usualServerlessDir).
	serverlessDir := os.Getenv("OVERRIDE_SANDBOX_DIR")
	if serverlessDir == "" {
		serverlessDir = usualServerlessDir
	}
	credsToken := HashAccessToken(accessToken)
	return &serverlessService{
		serverlessJs:  filepath.Join(serverlessDir, "sandbox.js"),
		serverlessDir: serverlessDir,
		credsDir:      GetCredentialDirectory(credsToken, usualServerlessDir),
		node:          filepath.Join(serverlessDir, nodeBin),
		userAgent:     fmt.Sprintf("doctl/%s serverless/%s", doctl.DoitVersion.String(), minServerlessVersion),
		client:        client,
		owClient:      nil,
		accessToken:   accessToken,
	}
}

// HashAccessToken converts a DO access token string into a shorter but suitably random string
// via hashing.  This is used to form part of the path for storing OpenWhisk credentials
func HashAccessToken(token string) string {
	hasher := sha1.New()
	hasher.Write([]byte(token))
	sha := hasher.Sum(nil)
	return hex.EncodeToString(sha[:4])
}

// InitWhisk is an on-demand initializer for the OpenWhisk client, called when that client
// is needed.
func initWhisk(s *serverlessService) error {
	if s.owClient != nil {
		return nil
	}
	var config *whisk.Config
	if s.owConfig != nil {
		config = s.owConfig
	} else {
		err := s.CheckServerlessStatus()
		if err != nil {
			return err
		}
		creds, err := s.ReadCredentials()
		if err != nil {
			return err
		}
		credential := creds.Credentials[creds.APIHost][creds.Namespace]
		config = &whisk.Config{Host: creds.APIHost, AuthToken: credential.Auth}
	}
	client, err := whisk.NewClient(http.DefaultClient, config)
	if err != nil {
		return err
	}
	s.owClient = client
	return nil
}

// SetEffectiveCredentials is used in low-level scenarios when we want to bypass normal credentialing.
// For example, doing things to serverless clusters that are not yet in full production.
func (s *serverlessService) SetEffectiveCredentials(auth string, apihost string) {
	s.owConfig = &whisk.Config{Host: apihost, AuthToken: auth}
	s.owClient = nil // ensure fresh initialization next time
}

func (s *serverlessService) CheckServerlessStatus() error {
	_, err := os.Stat(s.serverlessDir)
	if os.IsNotExist(err) {
		return ErrServerlessNotInstalled
	}
	if !serverlessUptodate(s.serverlessDir) {
		return ErrServerlessNeedsUpgrade
	}
	if !isServerlessConnected(s.credsDir) {
		return ErrServerlessNotConnected
	}
	return nil
}

// InstallServerless is the common subroutine for both serverless install and serverless upgrade
func (s *serverlessService) InstallServerless(leafCredsDir string, upgrading bool) error {
	serverlessDir := s.serverlessDir

	// Make a temporary directory for use during the install.
	// Note: we don't let this be allocated in the system temporaries area because
	// that might be on a separate file system, meaning that the final install step
	// will require an additional copy rather than a simple rename.

	os.Mkdir(filepath.Dir(serverlessDir), 0700) // in case using config dir and it doesn't exist yet
	tmp, err := ioutil.TempDir(filepath.Dir(serverlessDir), "sbx-install")
	if err != nil {
		return err
	}

	// Download the nodejs tarball for this os and architecture
	fmt.Print("Downloading...")

	goos := runtime.GOOS
	arch := runtime.GOARCH
	nodeBin := "node"
	if arch == "amd64" {
		arch = "x64"
	}
	if arch == "386" {
		if goos == "linux" {
			return errors.New("serverless support is not available for 32-bit linux")
		}
		arch = "x86"
	}
	if goos == "windows" {
		goos = "win"
		nodeBin = "node.exe"
	}

	var (
		nodeURL      string
		nodeFileName string
		nodeDir      string
	)

	// Download nodejs only if necessary
	if !upgrading || !canReuseNode(serverlessDir, nodeBin) {
		nodeDir = fmt.Sprintf("node-%s-%s-%s", nodeVersion, goos, arch)
		nodeURL = fmt.Sprintf("https://nodejs.org/dist/%s/%s.tar.gz", nodeVersion, nodeDir)
		nodeFileName = filepath.Join(tmp, "node-install.tar.gz")

		if goos == "win" {
			nodeURL = fmt.Sprintf("https://nodejs.org/dist/%s/%s.zip", nodeVersion, nodeDir)
			nodeFileName = filepath.Join(tmp, "node-install.zip")
		}

		err = download(nodeURL, nodeFileName)
		if err != nil {
			return err
		}
	}

	// Download the fat tarball with the nim CLI, deployer, and sandbox bridge
	// TODO do these need to be arch-specific?  Currently assuming not.
	URL := fmt.Sprintf("https://do-serverless-tools.nyc3.digitaloceanspaces.com/doctl-sandbox-%s.tar.gz",
		GetMinServerlessVersion())
	sandboxFileName := filepath.Join(tmp, "doctl-sandbox.tar.gz")
	err = download(URL, sandboxFileName)
	if err != nil {
		return err
	}

	// Exec the Extract utility at least once to unpack the fat tarball and possibly a second time if
	// node was downloaded.  If node was not downloaded, just move the existing binary into place.
	fmt.Print("Unpacking...")
	err = extract.Extract(sandboxFileName, tmp)
	if err != nil {
		return err
	}

	if nodeFileName != "" {
		err = extract.Extract(nodeFileName, tmp)
		if err != nil {
			return err
		}
	}

	// Move artifacts to final location
	fmt.Print("Installing...")
	srcPath := filepath.Join(tmp, "sandbox")
	if upgrading {
		// Preserve credentials by moving them from target (which will be replaced) to source.
		err = PreserveCreds(leafCredsDir, srcPath, serverlessDir)
		if err != nil {
			return err
		}
		// Preserve existing node if necessary
		if nodeFileName == "" {
			// Node was not downloaded
			err = moveExistingNode(serverlessDir, srcPath, nodeBin)
			if err != nil {
				return err
			}
		}
	} else {
		// Make new empty credentials directory
		emptyCreds := filepath.Join(srcPath, credsDir)
		err = os.MkdirAll(emptyCreds, 0700)
		if err != nil {
			return nil
		}

		// Create the sandbox directory if necessary.
		err := os.MkdirAll(serverlessDir, 0755)
		if err != nil {
			return err
		}
	}
	// Remove former serverlessDir before moving in the new one
	err = os.RemoveAll(serverlessDir)
	if err != nil {
		return err
	}
	err = os.Rename(srcPath, serverlessDir)
	if err != nil {
		return err
	}

	if nodeFileName != "" {
		if goos == "win" {
			srcPath = filepath.Join(tmp, nodeDir, nodeBin)
		} else {
			// Additional nesting in non-windows case
			srcPath = filepath.Join(tmp, nodeDir, "bin", nodeBin)
		}
		destPath := filepath.Join(serverlessDir, nodeBin)
		err = os.Rename(srcPath, destPath)
		if err != nil {
			return err
		}
	}
	// Clean up temp directory
	fmt.Print("Cleaning up...")
	os.RemoveAll(tmp) // Best effort, ignore error
	fmt.Println("\nDone")
	return nil
}

// Cmd builds an *exec.Cmd for calling into the sandbox plugin.
func (s *serverlessService) Cmd(command string, args []string) (*exec.Cmd, error) {
	args = append([]string{s.serverlessJs, command}, args...)
	cmd := exec.Command(s.node, args...)
	cmd.Env = append(os.Environ(), "NIMBELLA_DIR="+s.credsDir, "NIM_USER_AGENT="+s.userAgent, "DO_API_KEY="+s.accessToken)
	// If DEBUG is specified, we need to open up stderr for that stream.  The stdout stream
	// will continue to work for returning structured results.
	if os.Getenv("DEBUG") != "" {
		cmd.Stderr = os.Stderr
	}
	return cmd, nil
}

// Exec executes an *exec.Cmd and captures its output in a ServerlessOutput.
func (s *serverlessService) Exec(cmd *exec.Cmd) (ServerlessOutput, error) {
	output, err := cmd.Output()
	if err != nil {
		// Ignore "errors" that are just non-zero exit.  The
		// serverless uses this as a secondary indicator but the output
		// is still trustworthy (and includes error information inline)
		if _, ok := err.(*exec.ExitError); !ok {
			// Real error of some sort
			return ServerlessOutput{}, err
		}
	}
	var result ServerlessOutput
	err = json.Unmarshal(output, &result)
	if err != nil {
		return ServerlessOutput{}, err
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
func (s *serverlessService) Stream(cmd *exec.Cmd) error {

	return cmd.Run()
}

// GetServerlessNamespace returns the credentials of the one serverless namespace assigned to
// the invoking doctl context.
func (s *serverlessService) GetServerlessNamespace(ctx context.Context) (ServerlessCredentials, error) {
	path := "v2/functions/sandbox"
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return ServerlessCredentials{}, err
	}
	return executeNamespaceRequest(ctx, s, req)
}

// executeNamespaceRequest executes a valid http.Request object where the request is expected
// to return a NamespaceResponse.  The response is converted to ServerlessCredentials.  The request
// may represent the (new) 'namespaces/<name>' GET API, the (legacy) 'sandbox' POST API, or
// a namespace creation.
// The legacy API will continue to be used by some users until feature-flipper protection is removed
// from the new one.
func executeNamespaceRequest(ctx context.Context, s *serverlessService, req *http.Request) (ServerlessCredentials, error) {
	decoded := new(NamespaceResponse)
	_, err := s.client.Do(ctx, req, decoded)
	if err != nil {
		return ServerlessCredentials{}, err
	}
	host := assignAPIHost(decoded.Namespace.APIHost, decoded.Namespace.Namespace)
	credential := ServerlessCredential{Auth: decoded.Namespace.UUID + ":" + decoded.Namespace.Key}
	namespace := decoded.Namespace.Namespace
	ans := ServerlessCredentials{
		APIHost:     host,
		Namespace:   namespace,
		Label:       decoded.Namespace.Label,
		Credentials: map[string]map[string]ServerlessCredential{host: {namespace: credential}},
	}
	return ans, nil
}

// ListNamespaces obtains the full list of namespaces that belong to the requesting account
func (s *serverlessService) ListNamespaces(ctx context.Context) (NamespaceListResponse, error) {
	path := "v2/functions/namespaces"
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return NamespaceListResponse{}, err
	}
	decoded := new(NamespaceListResponse)
	_, err = s.client.Do(ctx, req, decoded)
	if err != nil {
		return NamespaceListResponse{}, err
	}
	return removeAppNamespaces(*decoded), nil
}

// removeAppNamespaces modifies a NamespaceListResponse to exclude namespaces prefixed by ap-.
// Those are supposed to be managed by App Platform and should not be available to doctl serverless
// for connection or modification of any kind.  This is intended to be temporary because the filtering
// really should be done within the API.
func removeAppNamespaces(input NamespaceListResponse) NamespaceListResponse {
	newList := []OutputNamespace{}
	for _, ns := range input.Namespaces {
		if strings.HasPrefix(ns.Namespace, "ap-") {
			continue
		}
		newList = append(newList, ns)
	}
	return NamespaceListResponse{Namespaces: newList}
}

// GetNamespace obtains the credentials of a specific namespace, given its name
func (s *serverlessService) GetNamespace(ctx context.Context, name string) (ServerlessCredentials, error) {
	path := "v2/functions/namespaces/" + name
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ServerlessCredentials{}, err
	}
	return executeNamespaceRequest(ctx, s, req)
}

// GetNamespaceFromCluster obtains the namespace that uniquely owns a valid combination of API host and "auth"
// (uuid:key).  This can be used to connect to clusters not known to the portal (e.g. dev clusters) or simply
// to check that credentials are valid.
func (s *serverlessService) GetNamespaceFromCluster(APIhost string, auth string) (string, error) {
	// We do not use the shared client in serverlessService for this because it uses the stored
	// credentials, not the passed ones.
	config := whisk.Config{Host: APIhost, AuthToken: auth}
	client, err := whisk.NewClient(http.DefaultClient, &config)
	if err != nil {
		return "", err
	}
	ns, _, err := client.Namespaces.List()
	if err != nil {
		return "", err
	}
	if len(ns) != 1 {
		return "", fmt.Errorf("unexpected response when validating apihost and auth")
	}
	return ns[0].Name, nil
}

// CreateNamespace creates a new namespace and returns its credentials, given a label and region
func (s *serverlessService) CreateNamespace(ctx context.Context, label string, region string) (ServerlessCredentials, error) {
	reqBody := newNamespaceRequest{Namespace: inputNamespace{Label: label, Region: region}}
	path := "v2/functions/namespaces"
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return ServerlessCredentials{}, err
	}
	return executeNamespaceRequest(ctx, s, req)
}

// DeleteNamespace deletes a namespace by name
func (s *serverlessService) DeleteNamespace(ctx context.Context, name string) error {
	path := "v2/functions/namespaces/" + name
	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	_, err = s.client.Do(ctx, req, nil)
	return err
}

func (s *serverlessService) CleanNamespace() error {
	// Deletes all triggers
	ctx := context.TODO()
	triggers, err := s.ListTriggers(ctx, "")

	// Intentionally ignore errors when listing triggers, the trigger API is behind a
	// feature flag and may will return an error for users not enabled.
	if err == nil {
		for _, trig := range triggers {
			err = s.DeleteTrigger(ctx, trig.Name)
			if err != nil {
				return err
			}
		}
	}

	// Deletes all functions
	fns, err := s.ListFunctions("", 0, 200)
	if err != nil {
		return err
	}

	for _, fn := range fns {
		name := fn.Name
		pkg := strings.Split(fn.Namespace, "/")
		if len(pkg) == 2 {
			name = pkg[1] + "/" + fn.Name
		}

		// All triggers for the namespace are deleted above so we don't need to remove the trigger for each individual function.
		err := s.DeleteFunction(name, false)
		if err != nil {
			return err
		}
	}

	// Now delete all packages.  Since the functions are presumably gone, the packages can be deleted non-recursively.
	pkgs, err := s.ListPackages()
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		err := s.DeletePackage(pkg.Name, false)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetHostInfo returns the HostInfo structure of the provided API host
func (s *serverlessService) GetHostInfo(APIHost string) (ServerlessHostInfo, error) {
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

// GetFunction returns the metadata and optionally the code of a deployer function
func (s *serverlessService) GetFunction(name string, fetchCode bool) (whisk.Action, []FunctionParameter, error) {
	err := initWhisk(s)
	if err != nil {
		return whisk.Action{}, []FunctionParameter{}, err
	}
	action, resp, err := s.owClient.Actions.Get(name, fetchCode)
	if err != nil {
		return whisk.Action{}, []FunctionParameter{}, err
	}
	var parameters []FunctionParameter
	if resp != nil {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			reparse := FunctionParameterReparse{}
			err = json.Unmarshal(body, &reparse)
			if err != nil {
				return whisk.Action{}, []FunctionParameter{}, err
			}
			parameters = reparse.Parameters
		}
	}
	return *action, parameters, nil
}

// ListFunctions lists the functions of the connected namespace
func (s *serverlessService) ListFunctions(pkg string, skip int, limit int) ([]whisk.Action, error) {
	err := initWhisk(s)
	if err != nil {
		return []whisk.Action{}, err
	}
	if limit == 0 {
		limit = 30
	}
	options := &whisk.ActionListOptions{
		Skip:  skip,
		Limit: limit,
	}
	list, _, err := s.owClient.Actions.List(pkg, options)
	return list, err
}

// DeleteFunction removes a function from the namespace
func (s *serverlessService) DeleteFunction(name string, deleteTriggers bool) error {
	err := initWhisk(s)
	if err != nil {
		return err
	}

	if deleteTriggers {
		ctx := context.TODO()
		triggers, err := s.ListTriggers(ctx, name)

		// Intentionally ignore errors when listing triggers, the trigger API is behind a
		// feature flag and may will return an error for users not enabled.
		if err == nil {
			for _, trig := range triggers {
				err = s.DeleteTrigger(ctx, trig.Name)
				if err != nil {
					return err
				}
			}
		}
	}

	_, e := s.owClient.Actions.Delete(name)

	return e
}

// ListPackages lists the packages of the namespace
func (s *serverlessService) ListPackages() ([]whisk.Package, error) {
	err := initWhisk(s)
	if err != nil {
		return []whisk.Package{}, err
	}
	options := whisk.PackageListOptions{Limit: 200} // 200 is the max and we are also treating it as a safe-enough value here
	list, _, err := s.owClient.Packages.List(&options)
	return list, err
}

// DeletePackage removes a package from the namespace.
// If recursive is set to true, it will remove all functions in the package.
func (s *serverlessService) DeletePackage(name string, recursive bool) error {
	err := initWhisk(s)
	if err != nil {
		return err
	}

	if recursive {
		pkg, _, err := s.owClient.Packages.Get(name)
		if err != nil {
			return err
		}

		for _, fn := range pkg.Actions {
			funcName := name + "/" + fn.Name

			err = s.DeleteFunction(funcName, true)

			if err != nil {
				return err
			}
		}
	}

	_, err = s.owClient.Packages.Delete(name)

	return err
}

// InvokeFunction invokes a function via POST with authentication
func (s *serverlessService) InvokeFunction(name string, params interface{}, blocking bool, result bool) (interface{}, error) {
	var empty map[string]interface{}
	err := initWhisk(s)
	if err != nil {
		return empty, err
	}
	resp, _, err := s.owClient.Actions.Invoke(name, params, blocking, result)
	return resp, err
}

// InvokeFunctionViaWeb invokes a function via GET using its web URL (or error if not a web function)
func (s *serverlessService) InvokeFunctionViaWeb(name string, params interface{}) error {
	// Get the function so we can use its metadata in formulating the request
	theFunction, _, err := s.GetFunction(name, false)
	if err != nil {
		return err
	}
	// Check that it's a web function
	isWeb := false
	for _, annot := range theFunction.Annotations {
		if annot.Key == "web-export" {
			isWeb = true
			break
		}
	}
	if !isWeb {
		return fmt.Errorf("'%s' is not a web function", name)
	}
	// Formulate the invocation URL
	host, err := s.GetConnectedAPIHost()
	if err != nil {
		return err
	}
	nsParts := strings.Split(theFunction.Namespace, "/")
	namespace := nsParts[0]
	pkg := "default"
	if len(nsParts) > 1 {
		pkg = nsParts[1]
	}
	theURL := fmt.Sprintf("%s/api/v1/web/%s/%s/%s", host, namespace, pkg, theFunction.Name)
	// Add params, if any
	if params != nil {
		encoded := url.Values{}
		for key, val := range params.(map[string]interface{}) {
			stringVal, ok := val.(string)
			if !ok {
				return fmt.Errorf("the value of '%s' is not a string; web invocation is not possible", key)
			}
			encoded.Add(key, stringVal)
		}
		theURL += "?" + encoded.Encode()
	}
	return browser.OpenURL(theURL)
}

// ListActivations drives the OpenWhisk API for listing activations
func (s *serverlessService) ListActivations(options whisk.ActivationListOptions) ([]whisk.Activation, error) {
	empty := []whisk.Activation{}
	err := initWhisk(s)
	if err != nil {
		return empty, err
	}
	resp, _, err := s.owClient.Activations.List(&options)
	return resp, err
}

// GetActivationCount drives the OpenWhisk API for getting the total number of activations in namespace
func (s *serverlessService) GetActivationCount(options whisk.ActivationCountOptions) (whisk.ActivationCount, error) {
	err := initWhisk(s)
	empty := whisk.ActivationCount{}
	if err != nil {
		return empty, err
	}

	resp, _, err := s.owClient.Activations.Count(&options)
	if err != nil {
		return empty, err
	}
	return *resp, err
}

// GetActivation drives the OpenWhisk API getting an activation
func (s *serverlessService) GetActivation(id string) (whisk.Activation, error) {
	empty := whisk.Activation{}
	err := initWhisk(s)
	if err != nil {
		return empty, err
	}

	resp, _, err := s.owClient.Activations.Get(id)
	if err != nil {
		return empty, err
	}
	return *resp, err
}

// GetActivationLogs drives the OpenWhisk API getting the logs of an activation
func (s *serverlessService) GetActivationLogs(id string) (whisk.Activation, error) {
	empty := whisk.Activation{}
	err := initWhisk(s)
	if err != nil {
		return empty, err
	}

	resp, _, err := s.owClient.Activations.Logs(id)
	if err != nil {
		return empty, err
	}

	return *resp, err
}

// GetActivationResult drives the OpenWhisk API getting the result of an activation
func (s *serverlessService) GetActivationResult(id string) (whisk.Response, error) {
	empty := whisk.Response{}
	err := initWhisk(s)
	if err != nil {
		return empty, err
	}

	resp, _, err := s.owClient.Activations.Result(id)
	if err != nil {
		return empty, err
	}
	return *resp, err
}

// GetConnectedAPIHost retrieves the API host to which the service is currently connected
func (s *serverlessService) GetConnectedAPIHost() (string, error) {
	err := initWhisk(s)
	if err != nil {
		return "", err
	}
	return s.owClient.Config.Host, nil
}

// ReadProject takes the path where project lies and validates the project.yml.
// once project.yml is validated it reads the directory for all the files and sub-directory
// and returns the struct of the files
func (s *serverlessService) ReadProject(project *ServerlessProject, args []string) (ServerlessOutput, error) {
	err := readTopLevel(project)
	if err != nil {
		return ServerlessOutput{}, err
	}
	_, err = readProjectConfig(project.ConfigPath)
	if err != nil {
		return ServerlessOutput{}, err
	}
	return ServerlessOutput{}, fmt.Errorf("not implemented")
}

// WriteProject ...
func (s *serverlessService) WriteProject(project ServerlessProject) (string, error) {
	// TODO
	return "", nil
}

// ListTriggers lists the triggers in the connected namespace.  If 'fcn' is a non-empty
// string it is assumed to be the package-qualified name of a function and only the triggers
// of that function are listed.  If 'fcn' is empty all triggers are listed.
func (s *serverlessService) ListTriggers(ctx context.Context, fcn string) ([]ServerlessTrigger, error) {
	empty := []ServerlessTrigger{}
	err := s.CheckServerlessStatus()
	if err != nil {
		return empty, err
	}
	creds, err := s.ReadCredentials()
	if err != nil {
		return empty, err
	}
	path := fmt.Sprintf("v2/functions/namespaces/%s/triggers", creds.Namespace)
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return empty, err
	}
	decoded := new(ServerlessTriggerListResponse)
	_, err = s.client.Do(ctx, req, decoded)
	if err != nil {
		return empty, err
	}
	triggers := decoded.Triggers
	// The API does not filter by function; that is done here.
	if fcn != "" {
		filtered := []ServerlessTrigger{}
		for _, trigger := range triggers {
			if trigger.Function == fcn {
				filtered = append(filtered, trigger)
			}
		}
		triggers = filtered
	}
	return triggers, nil
}

// GetTrigger gets the contents of a trigger for display
func (s *serverlessService) GetTrigger(ctx context.Context, name string) (ServerlessTrigger, error) {
	empty := ServerlessTrigger{}
	err := s.CheckServerlessStatus()
	if err != nil {
		return empty, err
	}
	creds, err := s.ReadCredentials()
	if err != nil {
		return empty, err
	}
	path := fmt.Sprintf("v2/functions/namespaces/%s/triggers/%s", creds.Namespace, name)
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return empty, err
	}
	decoded := new(ServerlessTriggerGetResponse)
	_, err = s.client.Do(ctx, req, decoded)
	if err != nil {
		return empty, err
	}
	return decoded.Trigger, nil
}

func (s *serverlessService) UpdateTrigger(ctx context.Context, trigger string, opts *UpdateTriggerRequest) (ServerlessTrigger, error) {
	empty := ServerlessTrigger{}
	err := s.CheckServerlessStatus()
	if err != nil {
		return empty, err
	}
	creds, err := s.ReadCredentials()
	if err != nil {
		return empty, err
	}

	path := fmt.Sprintf("v2/functions/namespaces/%s/triggers/%s", creds.Namespace, trigger)
	req, err := s.client.NewRequest(ctx, http.MethodPut, path, opts)

	if err != nil {
		return empty, err
	}

	decoded := new(ServerlessTriggerGetResponse)
	_, err = s.client.Do(ctx, req, decoded)
	if err != nil {
		return empty, err
	}
	return decoded.Trigger, nil
}

// Delete Trigger deletes a trigger from the namespace (used when undeploying triggers explicitly,
// not part of a more general undeploy; when undeploying a function or the entire namespace we rely
// on the deployer to delete associated triggers).
func (s *serverlessService) DeleteTrigger(ctx context.Context, name string) error {
	creds, err := s.ReadCredentials()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("v2/functions/namespaces/%s/triggers/%s", creds.Namespace, name)
	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	_, err = s.client.Do(ctx, req, nil)
	return err
}

func readTopLevel(project *ServerlessProject) error {
	const (
		Config   = "project.yml"
		Packages = "packages"
	)
	files, err := ioutil.ReadDir(project.ProjectPath)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.Name() == Config && !f.IsDir() {
			project.ConfigPath = project.ProjectPath + "/" + f.Name()
		} else if f.Name() == Packages && f.IsDir() {
			project.Packages = project.ProjectPath + "/" + f.Name()
		} else if f.Name() == ".nimbella" || f.Name() == ".deployed" {
			// Ignore
		} else if f.Name() == ".env" && !f.IsDir() {
			project.Env = project.ProjectPath + "/" + f.Name()
		} else {
			project.Strays = append(project.Strays, project.ProjectPath+"/"+f.Name())
		}
	}
	return nil
}

func readProjectConfig(configPath string) (*ServerlessSpec, error) {
	spec := ServerlessSpec{}
	// reading config file content
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(configPath, ".json") {
		// unmarshal project.json
		err = json.Unmarshal([]byte(content), &spec)
		if err != nil {
			return nil, err
		}
	} else {
		// unmarshal project.yml
		err = yaml.Unmarshal([]byte(content), &spec)
		if err != nil {
			return nil, err
		}
	}
	err = validateConfig(&spec)
	if err != nil {
		return nil, err
	}
	return &spec, nil
}

func validateConfig(config *ServerlessSpec) error {
	forbiddenConfigs, err := ListForbiddenConfigs(config)

	if err != nil {
		return err
	}

	if len(forbiddenConfigs) > 0 {
		return fmt.Errorf("project.yml contains forbidden fields")
	}

	return nil
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
func (s *serverlessService) WriteCredentials(creds ServerlessCredentials) error {
	// Create the directory into which the file will be written.
	err := os.MkdirAll(s.credsDir, 0700)
	if err != nil {
		return err
	}
	// Write the credentials
	credsPath := filepath.Join(s.credsDir, CredentialsFile)
	bytes, err := json.MarshalIndent(&creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(credsPath, bytes, 0600)
}

// CredentialsPath simply returns the directory path where credentials are stored
func (s *serverlessService) CredentialsPath() string {
	return s.credsDir
}

// ReadCredentials reads the current serverless credentials from the appropriate 'creds' diretory
func (s *serverlessService) ReadCredentials() (ServerlessCredentials, error) {
	creds := ServerlessCredentials{}
	credsPath := filepath.Join(s.credsDir, CredentialsFile)
	bytes, err := os.ReadFile(credsPath)
	if err != nil {
		return creds, err
	}
	err = json.Unmarshal(bytes, &creds)
	return creds, err
}

// Determines whether the serverless support appears to be connected.  The purpose is
// to fail fast (when feasible) when it clearly is not connected.
// However, it is important not to add excessive overhead on each call (e.g.
// asking the plugin to validate credentials), so the test is not foolproof.
// It merely tests whether a credentials directory has been created for the
// current doctl access token and appears to have a credentials.json in it.
func isServerlessConnected(credsDir string) bool {
	credsFile := filepath.Join(credsDir, CredentialsFile)
	_, err := os.Stat(credsFile)
	// We used to test specifically for "not found" here but in fact any error is enough to
	// prevent connections from working.
	return err == nil
}

// serverlessUptodate answers whether the installed version of the serverless support is at least
// what is required by doctl
func serverlessUptodate(serverlessDir string) bool {
	return GetCurrentServerlessVersion(serverlessDir) >= GetMinServerlessVersion()
}

// GetCurrentServerlessVersion gets the version of the current plugin.
// To be called only when the plugin is known to exist.
// Returns "0" if the installed plugin pre-dates the versioning system
// Otherwise, returns the version string stored in the serverless directory.
func GetCurrentServerlessVersion(serverlessDir string) string {
	versionFile := filepath.Join(serverlessDir, "version")
	contents, err := ioutil.ReadFile(versionFile)
	if err != nil {
		return "0"
	}
	return string(contents)
}

// GetMinServerlessVersion returns the minServerlessVersion (allows the constant to be overridden via an environment variable)
func GetMinServerlessVersion() string {
	fromEnv := os.Getenv("minServerlessVersion")
	if fromEnv != "" {
		return fromEnv
	}
	return minServerlessVersion
}

// GetCredentialDirectory returns the directory in which credentials should be stored for a given
// CmdConfig.  The actual leaf directory is a function of the access token being used.  This ties
// serverless credentials to DO credentials
func GetCredentialDirectory(leafDir string, serverlessDir string) string {
	return filepath.Join(serverlessDir, credsDir, leafDir)
}

// Gets the version of the node binary in the serverless.  Determine if it is
// usable or whether it has to be upgraded.
func canReuseNode(serverlessDir string, nodeBin string) bool {
	fullNodeBin := filepath.Join(serverlessDir, nodeBin)
	cmd := exec.Command(fullNodeBin, "--version")
	result, err := cmd.Output()
	if err == nil {
		installed := strings.TrimSpace(string(result))
		return installed == nodeVersion
	}
	return false
}

// Moves the existing node binary from the serverless that contains it to the new serverless being
// staged during an upgrade.  This preserves it for reuse and avoids the need to download.
func moveExistingNode(existing string, staging string, nodeBin string) error {
	srcPath := filepath.Join(existing, nodeBin)
	destPath := filepath.Join(staging, nodeBin)
	return os.Rename(srcPath, destPath)
}

// Download a network file to a local file
func download(URL, targetFile string) error {
	response, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("received status code %d attempting to download from %s",
			response.StatusCode, URL)
	}
	file, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}
	return nil
}

// PreserveCreds preserves existing credentials in a serverless directory
// that is about to be replaced by moving them to the staging directory
// containing the replacement.
func PreserveCreds(leafDir string, stagingDir string, serverlessDir string) error {
	credPath := filepath.Join(serverlessDir, credsDir)
	relocPath := filepath.Join(stagingDir, credsDir)
	err := os.Rename(credPath, relocPath)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// There was no creds directory.  Check for legacy form and convert as part
	// of preserving.
	legacyCredPath := filepath.Join(serverlessDir, ".nimbella")
	err = os.MkdirAll(relocPath, 0700)
	if err != nil {
		return err
	}
	moveLegacyTo := GetCredentialDirectory(leafDir, stagingDir)
	return os.Rename(legacyCredPath, moveLegacyTo)
}

// ListForbiddenConfigs returns a list of forbidden config values in a project spec.
func ListForbiddenConfigs(serverlessProject *ServerlessSpec) ([]string, error) {
	var forbiddenConfigs []string

	// validate package-level configs
	for _, p := range serverlessProject.Packages {
		packageLevelForbiddenConfigs, err := validateProjectLevelFields(p)
		if err != nil {
			return nil, fmt.Errorf("validating package-level serverless configs: %w", err)
		}
		forbiddenConfigs = append(forbiddenConfigs, packageLevelForbiddenConfigs...)

		//validate function-level forbidden configs
		for _, a := range p.Functions {
			actionLevelForbiddenConfigs, err := validateFunctionLevelFields(a)
			if err != nil {
				return nil, fmt.Errorf("validating package-level serverless configs: %w", err)
			}
			forbiddenConfigs = append(forbiddenConfigs, actionLevelForbiddenConfigs...)
		}
	}
	return forbiddenConfigs, nil
}

// ListInvalidWebsecureValues returns a list of forbidden websecure values for an action in a project spec.
// a valid websecure value is any string other than "true"
func ListInvalidWebsecureValues(serverlessProject *ServerlessSpec) ([]string, error) {
	var invalidValues = []string{}

	for _, p := range serverlessProject.Packages {
		for _, f := range p.Functions {
			switch value := f.WebSecure.(type) {
			case string:
				if strings.ToLower(value) == "true" { /* "true" is not a valid value */
					invalidValues = append(invalidValues, fmt.Sprintf("function %s in package %s configures an invalid value for webSecure: %v", f.Name, p.Name, value))
				}
				// any other value is fine
			default: // bool or any other type
				/* "web-action" must be a string */
				invalidValues = append(invalidValues, fmt.Sprintf("function %s in package %s configures an invalid value for webSecure: %v", f.Name, p.Name, value))
			}
		}
	}

	return invalidValues, nil
}

// validate project-level forbidden configs
func validateProjectLevelFields(serverlessPackage *ServerlessPackage) ([]string, error) {
	var forbiddenConfigs []string

	if serverlessPackage.Shared {
		forbiddenConfigs = append(forbiddenConfigs, ForbiddenConfigShared)
	}

	if _, ok := serverlessPackage.Annotations[ForbiddenAnnotationProvideAPIKey]; ok {
		forbiddenConfigs = append(forbiddenConfigs, ForbiddenAnnotationProvideAPIKey)
	}

	if _, ok := serverlessPackage.Annotations[ForbiddenAnnotationRequireWhiskAuth]; ok {
		forbiddenConfigs = append(forbiddenConfigs, ForbiddenAnnotationRequireWhiskAuth)
	}

	return forbiddenConfigs, nil
}

// validate project-level forbidden configs
func validateFunctionLevelFields(serverlessAction *ServerlessFunction) ([]string, error) {
	var forbiddenConfigs []string

	if _, ok := serverlessAction.Annotations[ForbiddenAnnotationProvideAPIKey]; ok {
		forbiddenConfigs = append(forbiddenConfigs, ForbiddenAnnotationProvideAPIKey)
	}

	if _, ok := serverlessAction.Annotations[ForbiddenAnnotationRequireWhiskAuth]; ok {
		forbiddenConfigs = append(forbiddenConfigs, ForbiddenAnnotationRequireWhiskAuth)
	}

	return forbiddenConfigs, nil
}
