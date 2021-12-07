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

package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	dockerconf "github.com/docker/cli/cli/config"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/spf13/cobra"
	k8sapiv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

type dockerConfig struct {
	Auths map[string]struct {
		Auth string `json:"auth,omitempty"`
	} `json:"auths"`
}

const (
	// DOSecretOperatorAnnotation is the annotation key so that dosecret operator can do it's magic
	// and help users pull private images automatically in their DOKS clusters
	DOSecretOperatorAnnotation = "digitalocean.com/dosecret-identifier"

	oauthTokenRevokeEndpoint = "https://cloud.digitalocean.com/v1/oauth/revoke"
)

// Registry creates the registry command
func Registry() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "registry",
			Aliases: []string{"reg", "r"},
			Short:   "Display commands for working with container registries",
			Long:    "The subcommands of `doctl registry` create, manage, and allow access to your private container registry.",
		},
	}

	cmd.AddCommand(Repository())
	cmd.AddCommand(GarbageCollection())
	cmd.AddCommand(RegistryOptions())

	createRegDesc := "This command creates a new private container registry with the provided name."
	cmdRunRegistryCreate := CmdBuilder(cmd, RunRegistryCreate, "create <registry-name>",
		"Create a private container registry", createRegDesc, Writer)
	AddStringFlag(cmdRunRegistryCreate, doctl.ArgSubscriptionTier, "", "basic",
		"Subscription tier for the new registry. Possible values: see `doctl registry options subscription-tiers`", requiredOpt())

	getRegDesc := "This command retrieves details about a private container registry including its name and the endpoint used to access it."
	CmdBuilder(cmd, RunRegistryGet, "get", "Retrieve details about a container registry",
		getRegDesc, Writer, aliasOpt("g"), displayerType(&displayers.Registry{}))

	deleteRegDesc := "This command permanently deletes a private container registry and all of its contents."
	cmdRunRegistryDelete := CmdBuilder(cmd, RunRegistryDelete, "delete",
		"Delete a container registry", deleteRegDesc, Writer, aliasOpt("d", "del", "rm"))
	AddBoolFlag(cmdRunRegistryDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force registry delete")

	loginRegDesc := "This command logs in Docker so that pull and push commands to your private container registry will be authenticated."
	cmdRegistryLogin := CmdBuilder(cmd, RunRegistryLogin, "login", "Log in Docker to a container registry",
		loginRegDesc, Writer)
	AddIntFlag(cmdRegistryLogin, doctl.ArgRegistryExpirySeconds, "", 0,
		"The length of time the registry credentials will be valid for in seconds. By default, the credentials do not expire.")

	logoutRegDesc := "This command logs Docker out of the private container registry, revoking access to it."
	cmdRunRegistryLogout := CmdBuilder(cmd, RunRegistryLogout, "logout", "Log out Docker from a container registry",
		logoutRegDesc, Writer)
	AddStringFlag(cmdRunRegistryLogout, doctl.ArgRegistryAuthorizationServerEndpoint, "", oauthTokenRevokeEndpoint, "The endpoint of the OAuth authorization server used to revoke credentials on logout.")

	kubeManifestDesc := `This command outputs a YAML-formatted Kubernetes secret manifest that can be used to grant a Kubernetes cluster pull access to your private container registry.

By default, the secret manifest will be applied to all the namespaces for the Kubernetes cluster using the DOSecret operator. The DOSecret operator is available on clusters running version 1.15.12-do.2 or greater. For older clusters or to restrict the secret to a specific namespace, use the --namespace flag.

Redirect the command's output to a file to save the manifest for later use or pipe it directly to kubectl to create the secret in your cluster:

    doctl registry kubernetes-manifest | kubectl apply -f -
`
	cmdRunKubernetesManifest := CmdBuilder(cmd, RunKubernetesManifest, "kubernetes-manifest",
		"Generate a Kubernetes secret manifest for a registry.",
		kubeManifestDesc, Writer, aliasOpt("k8s"))
	AddStringFlag(cmdRunKubernetesManifest, doctl.ArgObjectName, "", "",
		"The secret name to create. Defaults to the registry name prefixed with \"registry-\"")
	AddStringFlag(cmdRunKubernetesManifest, doctl.ArgObjectNamespace, "",
		"kube-system", "The Kubernetes namespace to hold the secret")

	dockerConfigDesc := `This command outputs a JSON-formatted Docker configuration that can be used to configure a Docker client to authenticate with your private container registry. This configuration is useful for configuring third-party tools that need access to your registry. For configuring your local Docker client use "doctl registry login" instead, as it will preserve the configuration of any other registries you have authenticated to.

By default this command generates read-only credentials. Use the --read-write flag to generate credentials that can push. The configuration produced by this command contains a DigitalOcean API token that can be used to access your account, so be sure to keep it secret.`

	cmdRunDockerConfig := CmdBuilder(cmd, RunDockerConfig, "docker-config",
		"Generate a docker auth configuration for a registry",
		dockerConfigDesc, Writer, aliasOpt("config"))
	AddBoolFlag(cmdRunDockerConfig, doctl.ArgReadWrite, "", false,
		"Generate credentials that can push to your registry")
	AddIntFlag(cmdRunDockerConfig, doctl.ArgRegistryExpirySeconds, "", 0,
		"The length of time the registry credentials will be valid for in seconds. By default, the credentials do not expire.")

	return cmd
}

// Repository creates the repository sub-command
func Repository() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "repository",
			Aliases: []string{"repo", "r"},
			Short:   "Display commands for working with repositories in a container registry",
			Long:    "The subcommands of `doctl registry repository` help you command actions related to a repository.",
		},
	}

	listRepositoriesDesc := `This command retrieves information about repositories in a registry, including:
  - The repository name
  - The latest tag of the repository
  - The compressed size for the latest tag
  - The manifest digest for the latest tag
  - The last updated timestamp
`
	CmdBuilder(
		cmd,
		RunListRepositories, "list",
		"List repositories for a container registry", listRepositoriesDesc,
		Writer, aliasOpt("ls"), displayerType(&displayers.Repository{}),
		hiddenCmd(),
	)

	listRepositoriesV2Desc := `This command retrieves information about repositories in a registry, including:
  - The repository name
  - The latest manifest of the repository
  - The latest manifest's latest tag, if any
  - The number of tags in the repository
  - The number of manifests in the repository
`

	CmdBuilder(
		cmd,
		RunListRepositoriesV2, "list-v2",
		"List repositories for a container registry", listRepositoriesV2Desc,
		Writer, aliasOpt("ls2"), displayerType(&displayers.Repository{}),
	)

	listRepositoryTagsDesc := `This command retrieves information about tags in a repository, including:
  - The tag name
  - The compressed size
  - The manifest digest
  - The last updated timestamp
`
	CmdBuilder(
		cmd,
		RunListRepositoryTags, "list-tags <repository>",
		"List tags for a repository in a container registry", listRepositoryTagsDesc,
		Writer, aliasOpt("lt"), displayerType(&displayers.RepositoryTag{}),
	)

	deleteTagDesc := "This command permanently deletes one or more repository tags."
	cmdRunRepositoryDeleteTag := CmdBuilder(
		cmd,
		RunRepositoryDeleteTag,
		"delete-tag <repository> <tag>...",
		"Delete one or more container repository tags",
		deleteTagDesc,
		Writer,
		aliasOpt("dt"),
	)
	AddBoolFlag(cmdRunRepositoryDeleteTag, doctl.ArgForce, doctl.ArgShortForce, false, "Force tag deletion")

	listRepositoryManifests := `This command retrieves information about manifests in a repository, including:
  - The manifest digest
  - The compressed size
  - The uncompressed size
  - The last updated timestamp
  - The manifest tags
  - The manifest blobs (available in detailed output only)
`
	CmdBuilder(
		cmd,
		RunListRepositoryManifests, "list-manifests <repository>",
		"List manifests for a repository in a container registry", listRepositoryManifests,
		Writer, aliasOpt("lm"), displayerType(&displayers.RepositoryManifest{}),
	)

	deleteManifestDesc := "This command permanently deletes one or more repository manifests by digest."
	cmdRunRepositoryDeleteManifest := CmdBuilder(
		cmd,
		RunRepositoryDeleteManifest,
		"delete-manifest <repository> <manifest-digest>...",
		"Delete one or more container repository manifests by digest",
		deleteManifestDesc,
		Writer,
		aliasOpt("dm"),
	)
	AddBoolFlag(cmdRunRepositoryDeleteManifest, doctl.ArgForce, doctl.ArgShortForce, false, "Force manifest deletion")

	return cmd
}

// GarbageCollection creates the garbage-collection subcommand
func GarbageCollection() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "garbage-collection",
			Aliases: []string{"gc", "g"},
			Short:   "Display commands for garbage collection for a container registry",
			Long:    "The subcommands of `doctl registry garbage-collection` start a garbage collection, retrieve or cancel a currently-active garbage collection or list past garbage collections for a specified registry.",
		},
	}

	runStartGarbageCollectionDesc := "This command starts a garbage collection on a container registry. There can be only one active garbage collection at a time for a given registry."
	cmdStartGarbageCollection := CmdBuilder(
		cmd,
		RunStartGarbageCollection,
		"start",
		"Start garbage collection for a container registry",
		runStartGarbageCollectionDesc,
		Writer,
		aliasOpt("s"),
		displayerType(&displayers.GarbageCollection{}),
	)
	AddBoolFlag(cmdStartGarbageCollection, doctl.ArgGCIncludeUntaggedManifests, "", false,
		"Include untagged manifests in garbage collection.")
	AddBoolFlag(cmdStartGarbageCollection, doctl.ArgGCExcludeUnreferencedBlobs, "", false,
		"Exclude unreferenced blobs from garbage collection.")
	AddBoolFlag(cmdStartGarbageCollection, doctl.ArgForce, doctl.ArgShortForce, false, "Run garbage collection without confirmation prompt")

	gcInfoIncluded := `
  - UUID
  - Status
  - Registry Name
  - Created At
  - Updated At
  - Blobs Deleted
  - Freed Bytes
`

	runGetGarbageCollectionDesc := "This command retrieves the currently-active garbage collection for a container registry, if any active garbage collection exists. Information included about the registry includes:" + gcInfoIncluded
	CmdBuilder(
		cmd,
		RunGetGarbageCollection,
		"get-active",
		"Retrieve information about the currently-active garbage collection for a container registry",
		runGetGarbageCollectionDesc,
		Writer,
		aliasOpt("ga", "g"),
		displayerType(&displayers.GarbageCollection{}),
	)

	runListGarbageCollectionsDesc := "This command retrieves a list of past garbage collections for a registry. Information about each garbage collection includes:" + gcInfoIncluded
	CmdBuilder(
		cmd,
		RunListGarbageCollections,
		"list",
		"Retrieve information about past garbage collections for a container registry",
		runListGarbageCollectionsDesc,
		Writer,
		aliasOpt("ls", "l"),
		displayerType(&displayers.GarbageCollection{}),
	)

	runCancelGarbageCollectionDesc := "This command cancels the currently-active garbage collection for a container registry."
	CmdBuilder(
		cmd,
		RunCancelGarbageCollection,
		"cancel",
		"Cancel the currently-active garbage collection for a container registry",
		runCancelGarbageCollectionDesc,
		Writer,
		aliasOpt("c"),
	)

	return cmd
}

// RegistryOptions creates the registry options subcommand
func RegistryOptions() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "options",
			Aliases: []string{"opts", "o"},
			Short:   "List available container registry options",
			Long:    "This command lists options available when creating or updating a container registry.",
		},
	}

	tiersDesc := "List available container registry subscription tiers"
	CmdBuilder(cmd, RunRegistryOptionsTiers, "subscription-tiers", tiersDesc, tiersDesc, Writer, aliasOpt("tiers"))

	return cmd
}

// Registry Run Commands

// RunRegistryCreate creates a registry
func RunRegistryCreate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	name := c.Args[0]
	subscriptionTier, err := c.Doit.GetString(c.NS, doctl.ArgSubscriptionTier)
	if err != nil {
		return err
	}

	rs := c.Registry()

	rcr := &godo.RegistryCreateRequest{
		Name:                 name,
		SubscriptionTierSlug: subscriptionTier,
	}
	r, err := rs.Create(rcr)
	if err != nil {
		return err
	}

	return displayRegistries(c, *r)
}

// RunRegistryGet returns the registry
func RunRegistryGet(c *CmdConfig) error {
	reg, err := c.Registry().Get()
	if err != nil {
		return err
	}

	return displayRegistries(c, *reg)
}

// RunRegistryDelete delete the registry
func RunRegistryDelete(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirm("delete registry") != nil {
		return fmt.Errorf("operation aborted")
	}

	return c.Registry().Delete()
}

// store execCommand in a variable. Lets us override it while testing
var execCommand = exec.Command

// RunRegistryLogin logs in Docker to the registry
func RunRegistryLogin(c *CmdConfig) error {
	expirySeconds, err := c.Doit.GetInt(c.NS, doctl.ArgRegistryExpirySeconds)
	if err != nil {
		return err
	}
	regCredReq := godo.RegistryDockerCredentialsRequest{
		ReadWrite: true,
	}
	if expirySeconds != 0 {
		regCredReq.ExpirySeconds = godo.Int(expirySeconds)
	}

	fmt.Printf("Logging Docker in to %s\n", c.Registry().Endpoint())
	creds, err := c.Registry().DockerCredentials(&regCredReq)
	if err != nil {
		return err
	}

	var dc dockerConfig
	err = json.Unmarshal(creds.DockerConfigJSON, &dc)
	if err != nil {
		return err
	}

	// read the login credentials from the docker config
	for host, conf := range dc.Auths {
		// decode and split into username + password
		creds, err := base64.StdEncoding.DecodeString(conf.Auth)
		if err != nil {
			return err
		}

		splitCreds := strings.Split(string(creds), ":")
		if len(splitCreds) != 2 {
			return fmt.Errorf("got invalid docker credentials")
		}
		user, pass := splitCreds[0], splitCreds[1]

		authconfig := configtypes.AuthConfig{
			Username:      user,
			Password:      pass,
			ServerAddress: host,
		}

		cf := dockerconf.LoadDefaultConfigFile(os.Stderr)
		dockerCreds := cf.GetCredentialsStore(authconfig.ServerAddress)
		err = dockerCreds.Store(authconfig)
		if err != nil {
			_, isSnap := os.LookupEnv("SNAP")
			if os.IsPermission(err) && isSnap {
				warn("Using the doctl Snap? Grant access to the doctl:dot-docker plug to use this command with: sudo snap connect doctl:dot-docker")
				return err
			}

			return err
		}

		err = cf.Save()
		if err != nil {
			return err
		}
	}

	return nil
}

// RunKubernetesManifest prints a Kubernetes manifest that provides read/pull access to the registry
func RunKubernetesManifest(c *CmdConfig) error {
	secretName, err := c.Doit.GetString(c.NS, doctl.ArgObjectName)
	if err != nil {
		return err
	}
	secretNamespace, err := c.Doit.GetString(c.NS, doctl.ArgObjectNamespace)
	if err != nil {
		return err
	}

	// if no secret name supplied, use the registry name
	if secretName == "" {
		reg, err := c.Registry().Get()
		if err != nil {
			return err
		}
		secretName = "registry-" + reg.Name
	}

	// fetch docker config
	dockerCreds, err := c.Registry().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
		ReadWrite: false,
	})
	if err != nil {
		return err
	}
	annotations := map[string]string{}

	if secretNamespace == k8smetav1.NamespaceSystem {
		annotations[DOSecretOperatorAnnotation] = secretName
	}

	// create the manifest for the secret
	secret := &k8sapiv1.Secret{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:        secretName,
			Namespace:   secretNamespace,
			Annotations: annotations,
		},
		Type: k8sapiv1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": dockerCreds.DockerConfigJSON,
		},
	}

	serializer := k8sjson.NewSerializerWithOptions(
		k8sjson.DefaultMetaFactory, nil, nil,
		k8sjson.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)

	return serializer.Encode(secret, c.Out)
}

// RunDockerConfig generates credentials and prints a Docker config that can be
// used to authenticate a Docker client with the registry.
func RunDockerConfig(c *CmdConfig) error {
	readWrite, err := c.Doit.GetBool(c.NS, doctl.ArgReadWrite)
	if err != nil {
		return err
	}
	expirySeconds, err := c.Doit.GetInt(c.NS, doctl.ArgRegistryExpirySeconds)
	if err != nil {
		return err
	}
	regCredReq := godo.RegistryDockerCredentialsRequest{
		ReadWrite: readWrite,
	}
	if expirySeconds != 0 {
		regCredReq.ExpirySeconds = godo.Int(expirySeconds)
	}

	dockerCreds, err := c.Registry().DockerCredentials(&regCredReq)
	if err != nil {
		return err
	}

	_, err = c.Out.Write(append(dockerCreds.DockerConfigJSON, '\n'))
	return err
}

// RunRegistryLogout logs Docker out of the registry
func RunRegistryLogout(c *CmdConfig) error {
	endpoint, err := c.Doit.GetString(c.NS, doctl.ArgRegistryAuthorizationServerEndpoint)
	if err != nil {
		return err
	}

	server := c.Registry().Endpoint()
	fmt.Printf("Removing login credentials for %s\n", server)

	cf := dockerconf.LoadDefaultConfigFile(os.Stderr)
	dockerCreds := cf.GetCredentialsStore(server)
	authConfig, err := dockerCreds.Get(server)
	if err != nil {
		return err
	}

	err = dockerCreds.Erase(server)
	if err != nil {
		_, isSnap := os.LookupEnv("SNAP")
		if os.IsPermission(err) && isSnap {
			warn("Using the doctl Snap? Grant access to the doctl:dot-docker plug to use this command with: sudo snap connect doctl:dot-docker")
			return err
		}

		return err
	}

	return c.Registry().RevokeOAuthToken(authConfig.Password, endpoint)
}

// Repository Run Commands

// RunListRepositories lists repositories for the registry
func RunListRepositories(c *CmdConfig) error {
	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	repositories, err := c.Registry().ListRepositories(registry.Name)
	if err != nil {
		return err
	}

	return displayRepositories(c, repositories...)
}

// RunListRepositoriesV2 lists repositories for the registry
func RunListRepositoriesV2(c *CmdConfig) error {
	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	repositories, err := c.Registry().ListRepositoriesV2(registry.Name)
	if err != nil {
		return err
	}

	return displayRepositoriesV2(c, repositories...)
}

// RunListRepositoryTags lists tags for the repository in a registry
func RunListRepositoryTags(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	tags, err := c.Registry().ListRepositoryTags(registry.Name, c.Args[0])
	if err != nil {
		return err
	}

	return displayRepositoryTags(c, tags...)
}

// RunListRepositoryManifests lists manifests for the repository in a registry
func RunListRepositoryManifests(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	manifests, err := c.Registry().ListRepositoryManifests(registry.Name, c.Args[0])
	if err != nil {
		return err
	}

	return displayRepositoryManifests(c, manifests...)
}

// RunRepositoryDeleteTag deletes one or more repository tags
func RunRepositoryDeleteTag(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	repository := c.Args[0]
	tags := c.Args[1:]

	if !force && AskForConfirm(fmt.Sprintf("delete %d repository tag(s)", len(tags))) != nil {
		return fmt.Errorf("operation aborted")
	}

	var errors []string
	for _, tag := range tags {
		if err := c.Registry().DeleteTag(registry.Name, repository, tag); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete all repository tags: \n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// RunRepositoryDeleteManifest deletes one or more repository manifests by digest
func RunRepositoryDeleteManifest(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	registry, err := c.Registry().Get()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	repository := c.Args[0]
	digests := c.Args[1:]

	if !force && AskForConfirm(fmt.Sprintf("delete %d repository manifest(s) by digest (including associated tags)", len(digests))) != nil {
		return fmt.Errorf("operation aborted")
	}

	var errors []string
	for _, digest := range digests {
		if err := c.Registry().DeleteManifest(registry.Name, repository, digest); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete all repository manifests: \n%s", strings.Join(errors, "\n"))
	}

	return nil
}

func displayRegistries(c *CmdConfig, registries ...do.Registry) error {
	item := &displayers.Registry{
		Registries: registries,
	}
	return c.Display(item)
}

func displayRepositories(c *CmdConfig, repositories ...do.Repository) error {
	item := &displayers.Repository{
		Repositories: repositories,
	}
	return c.Display(item)
}

func displayRepositoriesV2(c *CmdConfig, repositories ...do.RepositoryV2) error {
	item := &displayers.RepositoryV2{
		Repositories: repositories,
	}
	return c.Display(item)
}

func displayRepositoryTags(c *CmdConfig, tags ...do.RepositoryTag) error {
	item := &displayers.RepositoryTag{
		Tags: tags,
	}
	return c.Display(item)
}

func displayRepositoryManifests(c *CmdConfig, manifests ...do.RepositoryManifest) error {
	item := &displayers.RepositoryManifest{
		Manifests: manifests,
	}
	return c.Display(item)
}

// Garbage Collection run commands

// RunStartGarbageCollection starts a garbage collection for the specified
// registry.
func RunStartGarbageCollection(c *CmdConfig) error {
	var registryName string
	// we anticipate supporting multiple registries in the future by allowing the
	// user to specify a registry argument on the command line but defaulting to
	// the default registry returned by c.Registry().Get()
	if len(c.Args) == 0 {
		var err error
		registry, err := c.Registry().Get()
		if err != nil {
			return fmt.Errorf("failed to get registry: %w", err)
		}
		registryName = registry.Name
	} else if len(c.Args) == 1 {
		registryName = c.Args[0]
	} else {
		return doctl.NewTooManyArgsErr(c.NS)
	}

	includeUntaggedManifests, err := c.Doit.GetBool(c.NS, doctl.ArgGCIncludeUntaggedManifests)
	if err != nil {
		return err
	}

	excludeUnreferencedBlobs, err := c.Doit.GetBool(c.NS, doctl.ArgGCExcludeUnreferencedBlobs)
	if err != nil {
		return err
	}

	gcStartRequest := &godo.StartGarbageCollectionRequest{}
	if includeUntaggedManifests && !excludeUnreferencedBlobs {
		gcStartRequest.Type = godo.GCTypeUntaggedManifestsAndUnreferencedBlobs
	} else if includeUntaggedManifests && excludeUnreferencedBlobs {
		gcStartRequest.Type = godo.GCTypeUntaggedManifestsOnly
	} else if !includeUntaggedManifests && !excludeUnreferencedBlobs {
		gcStartRequest.Type = godo.GCTypeUnreferencedBlobsOnly
	} else {
		return fmt.Errorf("incompatible combination of include-untagged-manifests and exclude-unreferenced-blobs flags")
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	msg := "run garbage collection -- this will put your registry in read-only mode until it finishes"

	if !force && AskForConfirm(msg) != nil {
		return errOperationAborted
	}

	gc, err := c.Registry().StartGarbageCollection(registryName, gcStartRequest)
	if err != nil {
		return err
	}

	return displayGarbageCollections(c, *gc)
}

// RunGetGarbageCollection gets the specified registry's currently-active
// garbage collection.
func RunGetGarbageCollection(c *CmdConfig) error {
	var registryName string
	// we anticipate supporting multiple registries in the future by allowing the
	// user to specify a registry argument on the command line but defaulting to
	// the default registry returned by c.Registry().Get()
	if len(c.Args) == 0 {
		var err error
		registry, err := c.Registry().Get()
		if err != nil {
			return fmt.Errorf("failed to get registry: %w", err)
		}
		registryName = registry.Name
	} else if len(c.Args) == 1 {
		registryName = c.Args[0]
	} else {
		return doctl.NewTooManyArgsErr(c.NS)
	}

	gc, err := c.Registry().GetGarbageCollection(registryName)
	if err != nil {
		return err
	}

	return displayGarbageCollections(c, *gc)
}

// RunListGarbageCollections gets the specified registry's currently-active
// garbage collection.
func RunListGarbageCollections(c *CmdConfig) error {
	var registryName string
	// we anticipate supporting multiple registries in the future by allowing the
	// user to specify a registry argument on the command line but defaulting to
	// the default registry returned by c.Registry().Get()
	if len(c.Args) == 0 {
		var err error
		registry, err := c.Registry().Get()
		if err != nil {
			return fmt.Errorf("failed to get registry: %w", err)
		}
		registryName = registry.Name
	} else if len(c.Args) == 1 {
		registryName = c.Args[0]
	} else {
		return doctl.NewTooManyArgsErr(c.NS)
	}

	gcs, err := c.Registry().ListGarbageCollections(registryName)
	if err != nil {
		return err
	}

	return displayGarbageCollections(c, gcs...)
}

// RunCancelGarbageCollection gets the specified registry's currently-active
// garbage collection.
func RunCancelGarbageCollection(c *CmdConfig) error {
	var (
		registryName string
		gcUUID       string
	)

	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	} else if len(c.Args) == 1 { // <gc-uuid>
		gcUUID = c.Args[0]
	} else if len(c.Args) == 2 { // <registry-name> <gc-uuid>
		registryName = c.Args[0]
		gcUUID = c.Args[1]
	} else {
		return doctl.NewTooManyArgsErr(c.NS)
	}

	// we anticipate supporting multiple registries in the future by allowing the
	// user to specify a registry argument on the command line but defaulting to
	// the default registry returned by c.Registry().Get()
	if registryName == "" {
		registry, err := c.Registry().Get()
		if err != nil {
			return fmt.Errorf("failed to get registry: %w", err)
		}
		registryName = registry.Name
	}

	_, err := c.Registry().CancelGarbageCollection(registryName, gcUUID)
	if err != nil {
		return err
	}

	return nil
}

func displayGarbageCollections(c *CmdConfig, garbageCollections ...do.GarbageCollection) error {
	item := &displayers.GarbageCollection{
		GarbageCollections: garbageCollections,
	}
	return c.Display(item)
}

func RunRegistryOptionsTiers(c *CmdConfig) error {
	tiers, err := c.Registry().GetSubscriptionTiers()
	if err != nil {
		return err
	}

	item := &displayers.RegistrySubscriptionTiers{
		SubscriptionTiers: tiers,
	}
	return c.Display(item)
}
