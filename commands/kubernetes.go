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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeerrors "k8s.io/apimachinery/pkg/util/errors"
	clientauthentication "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	maxAPIFailures            = 5
	timeoutFetchingKubeconfig = 30 * time.Second

	defaultKubernetesNodeSize      = "s-1vcpu-2gb"
	defaultKubernetesNodeCount     = 3
	defaultKubernetesRegion        = "nyc1"
	defaultKubernetesLatestVersion = "latest"

	execCredentialKind = "ExecCredential"

	workflowDesc = `

A typical workflow is to use ` + "`" + `doctl kubernetes cluster create` + "`" + ` to create the cluster on DigitalOcean's infrastructure, then call ` + "`" + `doctl kubernetes cluster kubeconfig` + "`" + ` to configure ` + "`" + `kubectl` + "`" + ` to connect to the cluster. You are then able to use ` + "`" + `kubectl` + "`" + ` to create and manage workloads.`
	optionsDesc = `

The commands under ` + "`" + `doctl kubernetes options` + "`" + ` retrieve values used while creating clusters, such as the list of regions where cluster creation is supported.`
)

var getCurrentAuthContextFn = defaultGetCurrentAuthContextFn

func defaultGetCurrentAuthContextFn() string {
	if Context != "" {
		return Context
	}
	if authContext := viper.GetString("context"); authContext != "" {
		return authContext
	}
	return doctl.ArgDefaultContext
}

func errNoClusterByName(name string) error {
	return fmt.Errorf("no cluster goes by the name %q", name)
}

func errAmbiguousClusterName(name string, ids []string) error {
	return fmt.Errorf("many clusters go by the name %q, they have the following IDs: %v", name, ids)
}

func errNoPoolByName(name string) error {
	return fmt.Errorf("No node pool goes by the name %q", name)
}

func errAmbiguousPoolName(name string, ids []string) error {
	return fmt.Errorf("Many node pools go by the name %q, they have the following IDs: %v", name, ids)
}

func errNoClusterNodeByName(name string) error {
	return fmt.Errorf("No node goes by the name %q", name)
}

func errAmbiguousClusterNodeName(name string, ids []string) error {
	return fmt.Errorf("Many nodes go by the name %q, they have the following IDs: %v", name, ids)
}

// Kubernetes creates the kubernetes command.
func Kubernetes() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "kubernetes",
			Aliases: []string{"kube", "k8s", "k"},
			Short:   "Displays commands to manage Kubernetes clusters and configurations",
			Long:    "The commands under `doctl kubernetes` are for managing Kubernetes clusters and viewing configuration options relating to clusters." + workflowDesc + optionsDesc,
		},
	}

	cmd.AddCommand(kubernetesCluster())
	cmd.AddCommand(kubernetesOptions())
	cmd.AddCommand(kubernetesOneClicks())

	return cmd
}

// KubeconfigProvider allows a user to read from a remote and local Kubeconfig, and write to a
// local Kubeconfig.
type KubeconfigProvider interface {
	Remote(kube do.KubernetesService, clusterID string, expirySeconds int) (*clientcmdapi.Config, error)
	Local() (*clientcmdapi.Config, error)
	Write(config *clientcmdapi.Config) error
	ConfigPath() string
}

type kubeconfigProvider struct {
	pathOptions *clientcmd.PathOptions
}

// Remote returns the kubeconfig for the cluster with the given ID from DOKS.
func (p *kubeconfigProvider) Remote(kube do.KubernetesService, clusterID string, expirySeconds int) (*clientcmdapi.Config, error) {
	var kubeconfig []byte
	var err error
	if expirySeconds > 0 {
		kubeconfig, err = kube.GetKubeConfigWithExpiry(clusterID, int64(expirySeconds))
	} else {
		kubeconfig, err = kube.GetKubeConfig(clusterID)
	}
	if err != nil {
		return nil, err
	}

	return clientcmd.Load(kubeconfig)
}

// Local reads the kubeconfig from the user's local kubeconfig file.
func (p *kubeconfigProvider) Local() (*clientcmdapi.Config, error) {
	config, err := p.pathOptions.GetStartingConfig()
	if err != nil {
		if a, ok := err.(kubeerrors.Aggregate); ok {
			_, isSnap := os.LookupEnv("SNAP")

			for _, err := range a.Errors() {
				// this should NOT be a contains check but they are formatting the
				// error without implementing an unwrap (so the original permission
				// error type is lost).
				if strings.Contains(err.Error(), "permission denied") && isSnap {
					warn("Using the doctl Snap? Grant access to the doctl:kube-config plug to use this command with: sudo snap connect doctl:kube-config")
					return nil, err
				}

			}
		}

		return nil, err
	}

	return config, nil
}

// Write either writes to or updates an existing local kubeconfig file.
func (p *kubeconfigProvider) Write(config *clientcmdapi.Config) error {
	err := clientcmd.ModifyConfig(p.pathOptions, *config, false)
	if err != nil {
		_, ok := os.LookupEnv("SNAP")

		if os.IsPermission(err) && ok {
			warn("Using the doctl Snap? Grant access to the doctl:kube-config plug to use this command with: sudo snap connect doctl:kube-config")
		}

		return err
	}

	return nil
}

func (p *kubeconfigProvider) ConfigPath() string {
	path := p.pathOptions.GetDefaultFilename()

	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		if _, ok := os.LookupEnv("SNAP"); ok {
			warn("Using the doctl Snap? Please create the directory: %q before trying again", filepath.Dir(path))
		}
	}

	return path
}

// KubernetesCommandService is used to execute Kubernetes commands.
type KubernetesCommandService struct {
	KubeconfigProvider KubeconfigProvider
}

func kubernetesCommandService() *KubernetesCommandService {
	return &KubernetesCommandService{
		KubeconfigProvider: &kubeconfigProvider{
			pathOptions: clientcmd.NewDefaultPathOptions(),
		},
	}
}

func kubernetesCluster() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "cluster",
			Aliases: []string{"clusters", "c"},
			Short:   "Display commands for managing Kubernetes clusters",
			Long:    "The commands under `doctl kubernetes cluster` are for the management of Kubernetes clusters." + workflowDesc,
		},
	}

	k8sCmdService := kubernetesCommandService()

	cmd.AddCommand(kubernetesKubeconfig())

	cmd.AddCommand(kubernetesNodePools())

	cmd.AddCommand(kubernetesRegistryIntegration())

	nodePoolDetails := `- A list of node pools available inside the cluster`
	clusterDetails := `

- A unique ID for the cluster
- A human-readable name for the cluster
- The slug identifier for the region where the Kubernetes cluster is located.
- The slug identifier for the version of Kubernetes used for the cluster. If set to a minor version (e.g. ` + "`" + `1.14` + "`" + `), the latest version within it will be used (e.g. ` + "`" + `1.14.6-do.1` + "`" + `); if set to ` + "`" + `latest` + "`" + `, the latest published version will be used.
- A boolean value indicating whether the cluster will be automatically upgraded to new patch releases during its maintenance window.
- An object containing a "state" attribute whose value is set to a string indicating the current status of the node. Potential values include ` + "`" + `running` + "`" + `, ` + "`" + `provisioning` + "`" + `, and ` + "`" + `errored` + "`" + `.`
	CmdBuilder(cmd, k8sCmdService.RunKubernetesClusterGet, "get <id|name>", "Retrieve details about a Kubernetes cluster", `
This command retrieves the following details about a Kubernetes cluster: `+clusterDetails+`
- The base URL of the cluster's Kubernetes API server.
- The public IPv4 address of the cluster's Kubernetes API server.
- The range of IP addresses in the overlay network of the Kubernetes cluster in CIDR notation.
- The range of assignable IP addresses for services running in the Kubernetes cluster in CIDR notation.
- An array of tags applied to the Kubernetes cluster. All clusters are automatically tagged `+"`"+`k8s`+"`"+` and `+"`"+`k8s:$K8S_CLUSTER_ID`+"`"+`.
- A time value given in ISO8601 combined date and time format that represents when the Kubernetes cluster was created.
- A time value given in ISO8601 combined date and time format that represents when the Kubernetes cluster was last updated.
`+nodePoolDetails,
		Writer, aliasOpt("g"), displayerType(&displayers.KubernetesClusters{}))
	CmdBuilder(cmd, k8sCmdService.RunKubernetesClusterList, "list", "Retrieve the list of Kubernetes clusters for your account", `
This command retrieves the following details about all Kubernetes clusters that are on your account:`+clusterDetails+nodePoolDetails,
		Writer, aliasOpt("ls"), displayerType(&displayers.KubernetesClusters{}))
	CmdBuilder(cmd, k8sCmdService.RunKubernetesClusterGetUpgrades, "get-upgrades <id|name>",
		"Retrieve a list of available Kubernetes version upgrades", `
This command returns a list of slugs representing Kubernetes versions you can use with the specified cluster. You can use these values to upgrade your cluster with the `+"`"+`doctl kubernetes cluster upgrade`+"`"+` command.
`, Writer, aliasOpt("gu"))

	cmdKubeClusterCreate := CmdBuilder(cmd, k8sCmdService.RunKubernetesClusterCreate(defaultKubernetesNodeSize,
		defaultKubernetesNodeCount), "create <name>", "Create a Kubernetes cluster", `
Creates a Kubernetes cluster given the specified options, using the specified name. Before creating the cluster, you can use `+"`"+`doctl kubernetes options`+"`"+` to see possible values for the various configuration flags.

If no configuration flags are used, a three-node cluster with a single node pool will be created in the nyc1 region, using the latest Kubernetes version.

After creating a cluster, a configuration context will be added to kubectl and made active so that you can begin managing your new cluster immediately.`,
		Writer, aliasOpt("c"))
	AddStringFlag(cmdKubeClusterCreate, doctl.ArgRegionSlug, "", defaultKubernetesRegion,
		"Cluster region. Possible values: see `doctl kubernetes options regions`", requiredOpt())
	AddStringFlag(cmdKubeClusterCreate, doctl.ArgClusterVersionSlug, "", "latest",
		"Kubernetes version. Possible values: see `doctl kubernetes options versions`")
	AddStringFlag(cmdKubeClusterCreate, doctl.ArgClusterVPCUUID, "", "",
		"Kubernetes UUID. Must be the UUID of a valid VPC in the same region specified for the cluster.")
	AddBoolFlag(cmdKubeClusterCreate, doctl.ArgAutoUpgrade, "", false,
		"A boolean flag indicating whether the cluster will be automatically upgraded to new patch releases during its maintenance window (default false). To enable automatic upgrade, supply --auto-upgrade(=true).")
	AddBoolFlag(cmdKubeClusterCreate, doctl.ArgSurgeUpgrade, "", true,
		"Boolean specifying whether to enable surge-upgrade for the cluster")
	AddBoolFlag(cmdKubeClusterCreate, doctl.ArgHA, "", false,
		"A boolean flag indicating whether the cluster will be configured with a highly-available control plane (default false). To enable the HA control plane, supply --ha(=true).")
	AddStringSliceFlag(cmdKubeClusterCreate, doctl.ArgTag, "", nil,
		"Comma-separated list of tags to apply to the cluster, in addition to the default tags of `k8s` and `k8s:$K8S_CLUSTER_ID`.")
	AddStringFlag(cmdKubeClusterCreate, doctl.ArgSizeSlug, "",
		defaultKubernetesNodeSize,
		"Machine size to use when creating nodes in the default node pool (incompatible with --"+doctl.ArgClusterNodePool+"). Possible values: see `doctl kubernetes options sizes`")
	AddStringSliceFlag(cmdKubeClusterCreate, doctl.ArgOneClicks, "", nil, "Comma-separated list of 1-Click Applications to install on the kubernetes cluster. To see a list of 1-Click Applications available run doctl kubernetes 1-click list")
	AddIntFlag(cmdKubeClusterCreate, doctl.ArgNodePoolCount, "",
		defaultKubernetesNodeCount,
		"Number of nodes in the default node pool (incompatible with --"+doctl.ArgClusterNodePool+")")
	AddStringSliceFlag(cmdKubeClusterCreate, doctl.ArgClusterNodePool, "", nil,
		`Comma-separated list of node pools, defined using semicolon-separated configuration values and surrounded by quotes (incompatible with --`+doctl.ArgSizeSlug+` and --`+doctl.ArgNodePoolCount+`)
Format: `+"`"+`"name=your-name;size=size_slug;count=5;tag=tag1;tag=tag2;label=key1=value1;label=key2=label2;taint=key1=value1:NoSchedule;taint=key2:NoExecute"`+"`"+` where:

- `+"`"+`name`+"`"+`: Name of the node pool, which must be unique in the cluster
- `+"`"+`size`+"`"+`: Machine size of the nodes to use. Possible values: see `+"`"+`doctl kubernetes options sizes`+"`"+`.
- `+"`"+`count`+"`"+`: Number of nodes to create.
- `+"`"+`tag`+"`"+`: Comma-separated list of tags to apply to nodes in the pool
- `+"`"+`label`+"`"+`: Label in key=value notation; repeat to add multiple labels at once.
- `+"`"+`taint`+"`"+`: Taint in key[=value]:effect notation; repeat to add multiple taints at once.
- `+"`"+`auto-scale`+"`"+`: Boolean defining whether to enable cluster auto-scaling on the node pool.
- `+"`"+`min-nodes`+"`"+`: Minimum number of nodes that can be auto-scaled to.
- `+"`"+`max-nodes`+"`"+`: Maximum number of nodes that can be auto-scaled to.`)

	AddBoolFlag(cmdKubeClusterCreate, doctl.ArgClusterUpdateKubeconfig, "", true,
		"Boolean that specifies whether to add a configuration context for the new cluster to your kubectl")
	AddBoolFlag(cmdKubeClusterCreate, doctl.ArgCommandWait, "", true,
		"Boolean that specifies whether to wait for cluster creation to complete before returning control to the terminal")
	AddBoolFlag(cmdKubeClusterCreate, doctl.ArgSetCurrentContext, "", true,
		"Boolean that specifies whether to set the current kubectl context to that of the new cluster")
	AddStringFlag(cmdKubeClusterCreate, doctl.ArgMaintenanceWindow, "", "any=00:00",
		"Sets the beginning of the four hour maintenance window for the cluster. Syntax is in the format: `day=HH:MM`, where time is in UTC. Day can be: `any`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday"+"`.")

	cmdKubeClusterUpdate := CmdBuilder(cmd, k8sCmdService.RunKubernetesClusterUpdate, "update <id|name>",
		"Update a Kubernetes cluster's configuration", `
This command updates the specified configuration values for the specified Kubernetes cluster. The cluster must be referred to by its name or ID, which you can retrieve by calling:

	doctl kubernetes cluster list`, Writer, aliasOpt("u"))
	AddStringFlag(cmdKubeClusterUpdate, doctl.ArgClusterName, "", "",
		"Specifies a new cluster name")
	AddStringSliceFlag(cmdKubeClusterUpdate, doctl.ArgTag, "", nil,
		"A comma-separated list of tags to apply to the cluster. Existing user-provided tags will be removed from the cluster if they are not specified.")
	AddBoolFlag(cmdKubeClusterUpdate, doctl.ArgAutoUpgrade, "", false,
		"A boolean flag indicating whether the cluster will be automatically upgraded to new patch releases during its maintenance window (default false). To enable automatic upgrade, supply --auto-upgrade(=true).")
	AddBoolFlag(cmdKubeClusterUpdate, doctl.ArgSurgeUpgrade, "", false,
		"Boolean specifying whether to enable surge-upgrade for the cluster")
	AddBoolFlag(cmdKubeClusterUpdate, doctl.ArgClusterUpdateKubeconfig, "",
		true, "Boolean specifying whether to update the cluster in your kubeconfig")
	AddBoolFlag(cmdKubeClusterUpdate, doctl.ArgSetCurrentContext, "", true,
		"Boolean specifying whether to set the current kubectl context to that of the new cluster")
	AddStringFlag(cmdKubeClusterUpdate, doctl.ArgMaintenanceWindow, "", "any=00:00",
		"Sets the beginning of the four hour maintenance window for the cluster. Syntax is in the format: 'day=HH:MM', where time is in UTC. Day can be: `any`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday"+"`.")

	cmdKubeClusterUpgrade := CmdBuilder(cmd, k8sCmdService.RunKubernetesClusterUpgrade,
		"upgrade <id|name>", "Upgrades a cluster to a new Kubernetes version", `

This command upgrades the specified Kubernetes cluster. By default, this will upgrade the cluster to the latest available release, but you can also specify any version listed for your cluster by using `+"`"+`doctl k8s get-upgrades`+"`"+`.`, Writer)
	AddStringFlag(cmdKubeClusterUpgrade, doctl.ArgClusterVersionSlug, "", "latest",
		`The desired Kubernetes version. Possible values: see `+"`"+`doctl k8s get-upgrades <cluster>`+"`"+`.
The special value `+"`"+`latest`+"`"+` will select the most recent patch version for your cluster's minor version.
For example, if a cluster is on 1.12.1 and upgrades are available to 1.12.3 and 1.13.1, 1.12.3 will be `+"`"+`latest`+"`"+`.`)

	cmdKubeClusterDelete := CmdBuilder(cmd, k8sCmdService.RunKubernetesClusterDelete,
		"delete <id|name>...", "Delete Kubernetes clusters ", `
This command deletes the specified Kubernetes clusters and the Droplets associated with them. To delete all other DigitalOcean resources created during the operation of the clusters, such as load balancers, volumes or volume snapshots, use the --dangerous flag.
`, Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdKubeClusterDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Boolean indicating whether to delete the cluster without a confirmation prompt")
	AddBoolFlag(cmdKubeClusterDelete, doctl.ArgClusterUpdateKubeconfig, "", true,
		"Boolean indicating whether to remove the deleted cluster from your kubeconfig")
	AddBoolFlag(cmdKubeClusterDelete, doctl.ArgDangerous, "", false,
		"Boolean indicating whether to delete the cluster's associated resources like load balancers, volumes and volume snapshots")

	cmdKubeClusterDeleteSelective := CmdBuilder(cmd, k8sCmdService.RunKubernetesClusterDeleteSelective,
		"delete-selective <id|name>", "Delete a Kubernetes cluster and selectively delete resources associated with it", `
This command deletes the specified Kubernetes cluster and droplets associated with it. It also deletes the specified associated resources. The associated resources supported for selective deletion are load balancers, volumes and volume snapshots.
`, Writer, aliasOpt("ds"))
	AddBoolFlag(cmdKubeClusterDeleteSelective, doctl.ArgForce, doctl.ArgShortForce, false,
		"Boolean indicating whether to delete the cluster without a confirmation prompt")
	AddBoolFlag(cmdKubeClusterDeleteSelective, doctl.ArgClusterUpdateKubeconfig, "", true,
		"Boolean indicating whether to remove the deleted cluster from your kubeconfig")
	AddStringSliceFlag(cmdKubeClusterDeleteSelective, doctl.ArgVolumeList, "", nil,
		"Comma-separated list of volume IDs or names for deletion")
	AddStringSliceFlag(cmdKubeClusterDeleteSelective, doctl.ArgVolumeSnapshotList, "", nil,
		"Comma-separated list of volume snapshot IDs or names for deletion")
	AddStringSliceFlag(cmdKubeClusterDeleteSelective, doctl.ArgLoadBalancerList, "", nil,
		"Comma-separated list of load balancer IDs or names for deletion")

	CmdBuilder(cmd, k8sCmdService.RunKubernetesClusterListAssociatedResources, "list-associated-resources <id|name>", "Retrieve DigitalOcean resources associated with a Kubernetes cluster", `
This command retrieves the following details:
- Volume IDs for volumes created by the DigitalOcean CSI driver
- Volume snapshot IDs for volume snapshots created by the DigitalOcean CSI driver.
- Load balancer IDs for load balancers managed by the Kubernetes cluster.`,
		Writer, aliasOpt("ar"), displayerType(&displayers.KubernetesAssociatedResources{}))

	return cmd
}

func kubernetesKubeconfig() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "kubeconfig",
			Aliases: []string{"kubecfg", "k8scfg", "config", "cfg"},
			Short:   "Display commands for managing your local kubeconfig",
			Long:    "The commands under `doctl kubernetes cluster kubeconfig` are used to manage Kubernetes cluster credentials on your local machine. The credentials are used as authentication contexts with `kubectl`, the Kubernetes command-line interface.",
		},
	}

	k8sCmdService := kubernetesCommandService()

	cmdShowConfig := CmdBuilder(cmd, k8sCmdService.RunKubernetesKubeconfigShow, "show <cluster-id|cluster-name>", "Show a Kubernetes cluster's kubeconfig YAML", `
This command prints out the raw YAML for the specified cluster's kubeconfig.	`, Writer, aliasOpt("p", "g"))
	AddIntFlag(cmdShowConfig, doctl.ArgKubeConfigExpirySeconds, "", 0,
		"The length of time the cluster credentials will be valid for in seconds. By default, the credentials expire after seven days.")

	execCredDesc := "INTERNAL: This hidden command is for printing a cluster's exec credential"
	cmdExecCredential := CmdBuilder(cmd, k8sCmdService.RunKubernetesKubeconfigExecCredential, "exec-credential <cluster-id>", execCredDesc, execCredDesc, Writer, hiddenCmd())
	AddStringFlag(cmdExecCredential, doctl.ArgVersion, "", "", "")

	cmdSaveConfig := CmdBuilder(cmd, k8sCmdService.RunKubernetesKubeconfigSave, "save <cluster-id|cluster-name>", "Save a cluster's credentials to your local kubeconfig", `
This command adds the credentials for the specified cluster to your local kubeconfig. After this, your kubectl installation can directly manage the specified cluster.
		`, Writer, aliasOpt("s"))
	AddBoolFlag(cmdSaveConfig, doctl.ArgSetCurrentContext, "", true, "Boolean indicating whether to set the current kubectl context to that of the new cluster")
	AddIntFlag(cmdSaveConfig, doctl.ArgKubeConfigExpirySeconds, "", 0,
		"The length of time the cluster credentials will be valid for in seconds. By default, the credentials are automatically renewed as needed.")
	AddStringFlag(cmdSaveConfig, doctl.ArgKubernetesAlias, "", "", "An alias for the cluster context name. Defaults to 'do-<region>-<cluster-name>'.")

	CmdBuilder(cmd, k8sCmdService.RunKubernetesKubeconfigRemove, "remove <cluster-id|cluster-name>", "Remove a cluster's credentials from your local kubeconfig", `
This command removes the specified cluster's credentials from your local kubeconfig. After running this command, you will not be able to use `+"`"+`kubectl`+"`"+` to interact with your cluster.
`, Writer, aliasOpt("d", "rm"))
	return cmd
}

func kubeconfigCachePath() string {
	return filepath.Join(defaultConfigHome(), "cache", "exec-credential")
}

func kubernetesNodePools() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "node-pool",
			Aliases: []string{"node-pools", "nodepool", "nodepools", "pool", "pools", "np", "p"},
			Short:   "Display commands for managing node pools",
			Long:    "The commands under `node-pool` are for performing actions on a Kubernetes cluster's node pools. You can use these commands to create or delete node pools, enable autoscaling for a node pool, and more.",
		},
	}

	k8sCmdService := kubernetesCommandService()

	CmdBuilder(cmd, k8sCmdService.RunKubernetesNodePoolGet, "get <cluster-id|cluster-name> <pool-id|pool-name>",
		"Retrieve information about a cluster's node pool", `
This command retrieves information about the specified node pool in the specified cluster, including:

- The node pool ID
- The machine size of the nodes (e.g. `+"`"+`s-1vcpu-2gb`+"`"+`)
- The number of nodes in the pool
- Tags applied to the node pool
- The names of the nodes

Specifying `+"`"+`--output=json`+"`"+` when calling this command will produce extra information about the individual nodes in the response, such as their IDs, status, creation time, and update time.
`, Writer, aliasOpt("g"),
		displayerType(&displayers.KubernetesNodePools{}))
	CmdBuilder(cmd, k8sCmdService.RunKubernetesNodePoolList, "list <cluster-id|cluster-name>",
		"List a cluster's node pools", `
This command retrieves information about the specified cluster's node pools, including:

- The node pool ID
- The machine size of the nodes (e.g. `+"`"+`s-1vcpu-2gb`+"`"+`)
- The number of nodes in the pool
- Tags applied to the node pool
- The names of the nodes

Specifying `+"`"+`--output=json`+"`"+` when calling this command will produce extra information about the individual nodes in the response, such as their IDs, status, creation time, and update time.
		`, Writer, aliasOpt("ls"),
		displayerType(&displayers.KubernetesNodePools{}))

	cmdKubeNodePoolCreate := CmdBuilder(cmd, k8sCmdService.RunKubernetesNodePoolCreate,
		"create <cluster-id|cluster-name>", "Create a new node pool for a cluster", `
This command creates a new node pool for the specified cluster. At a minimum, you'll need to specify the size of the nodes, and the number of nodes to place in the pool. You can also specify that you'd like to enable autoscaling and set minimum and maximum node poll sizes.
		`,
		Writer, aliasOpt("c"))
	AddStringFlag(cmdKubeNodePoolCreate, doctl.ArgNodePoolName, "", "",
		"Name of the node pool", requiredOpt())
	AddStringFlag(cmdKubeNodePoolCreate, doctl.ArgSizeSlug, "", "",
		"Size of the nodes in the node pool (To see possible values: call `doctl kubernetes options sizes`)", requiredOpt())
	AddIntFlag(cmdKubeNodePoolCreate, doctl.ArgNodePoolCount, "", 0,
		"The size of (number of nodes in) the node pool", requiredOpt())
	AddStringSliceFlag(cmdKubeNodePoolCreate, doctl.ArgTag, "", nil,
		"Tag to apply to the node pool; repeat to specify additional tags. An existing tag is removed from the node pool if it is not specified by any flag.")
	AddStringSliceFlag(cmdKubeNodePoolCreate, doctl.ArgKubernetesLabel, "", nil,
		"Label in key=value notation to apply to the node pool; repeat to specify additional labels. An existing label is removed from the node pool if it is not specified by any flag.")
	AddStringSliceFlag(cmdKubeNodePoolCreate, doctl.ArgKubernetesTaint, "", nil,
		"Taint in key[=value:]effect notation to apply to the node pool; repeat to specify additional taints. Set to the empty string \"\" to clear all taints. An existing taint is removed from the node pool if it is not specified by any flag.")
	AddBoolFlag(cmdKubeNodePoolCreate, doctl.ArgNodePoolAutoScale, "", false,
		"Boolean indicating whether to enable auto-scaling on the node pool")
	AddIntFlag(cmdKubeNodePoolCreate, doctl.ArgNodePoolMinNodes, "", 0,
		"Minimum number of nodes in the node pool when autoscaling is enabled")
	AddIntFlag(cmdKubeNodePoolCreate, doctl.ArgNodePoolMaxNodes, "", 0,
		"Maximum number of nodes in the node pool when autoscaling is enabled")

	cmdKubeNodePoolUpdate := CmdBuilder(cmd, k8sCmdService.RunKubernetesNodePoolUpdate,
		"update <cluster-id|cluster-name> <pool-id|pool-name>",
		"Update an existing node pool in a cluster", "This command updates the specified node pool in the specified cluster. You can update any value for which there is a flag.", Writer, aliasOpt("u"))
	AddStringFlag(cmdKubeNodePoolUpdate, doctl.ArgNodePoolName, "", "", "Name of the node pool")
	AddIntFlag(cmdKubeNodePoolUpdate, doctl.ArgNodePoolCount, "", 0,
		"The size of (number of nodes in) the node pool")
	AddStringSliceFlag(cmdKubeNodePoolUpdate, doctl.ArgTag, "", nil,
		"Tag to apply to the node pool; you can supply multiple `--tag` arguments to specify additional tags. Omitted tags will be removed from the node pool if the flag is specified.")
	AddStringSliceFlag(cmdKubeNodePoolUpdate, doctl.ArgKubernetesLabel, "", nil,
		"Label in key=value notation to apply to the node pool, repeat to add multiple labels at once. Omitted labels will be removed from the node pool if the flag is specified.")
	AddStringSliceFlag(cmdKubeNodePoolUpdate, doctl.ArgKubernetesTaint, "", nil,
		"Taint in key[=value:]effect notation to apply to the node pool, repeat to add multiple taints at once. Omitted taints will be removed from the node pool if the flag is specified.")
	AddBoolFlag(cmdKubeNodePoolUpdate, doctl.ArgNodePoolAutoScale, "", false,
		"Boolean indicating whether to enable auto-scaling on the node pool")
	AddIntFlag(cmdKubeNodePoolUpdate, doctl.ArgNodePoolMinNodes, "", 0,
		"Minimum number of nodes in the node pool when autoscaling is enabled")
	AddIntFlag(cmdKubeNodePoolUpdate, doctl.ArgNodePoolMaxNodes, "", 0,
		"Maximum number of nodes in the node pool when autoscaling is enabled")

	recycleDesc := "DEPRECATED: Use `replace-node`. Recycle nodes in a node pool"
	cmdKubeNodePoolRecycle := CmdBuilder(cmd, k8sCmdService.RunKubernetesNodePoolRecycle,
		"recycle <cluster-id|cluster-name> <pool-id|pool-name>", recycleDesc, recycleDesc, Writer, aliasOpt("r"), hiddenCmd())
	AddStringFlag(cmdKubeNodePoolRecycle, doctl.ArgNodePoolNodeIDs, "", "",
		"ID or name of the nodes in the node pool to recycle")

	cmdKubeNodePoolDelete := CmdBuilder(cmd, k8sCmdService.RunKubernetesNodePoolDelete,
		"delete <cluster-id|cluster-name> <pool-id|pool-name>",
		"Delete a node pool", `This command deletes the specified node pool in the specified cluster, which also removes all the nodes inside that pool. This action is irreversable.`, Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdKubeNodePoolDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "Delete node pool without confirmation prompt")

	cmdKubeNodeDelete := CmdBuilder(cmd, k8sCmdService.RunKubernetesNodeDelete, "delete-node <cluster-id|cluster-name> <pool-id|pool-name> <node-id>", "Delete a node", `
This command deletes the specified node, located in the specified node pool. By default this deletion will happen gracefully, and Kubernetes will drain the node of any pods before deleting it.
		`, Writer)
	AddBoolFlag(cmdKubeNodeDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the node without a confirmation prompt")
	AddBoolFlag(cmdKubeNodeDelete, "skip-drain", "", false, "Skip draining the node before deletion")

	cmdKubeNodeReplace := CmdBuilder(cmd, k8sCmdService.RunKubernetesNodeReplace, "replace-node <cluster-id|cluster-name> <pool-id|pool-name> <node-id>", "Replace node with a new one", `
This command deletes the specified node in the specified node pool, and then creates a new node in its place. This is useful if you suspect a node has entered an undesired state. By default the deletion will happen gracefully, and Kubernetes will drain the node of any pods before deleting it.
		`, Writer)
	AddBoolFlag(cmdKubeNodeReplace, doctl.ArgForce, doctl.ArgShortForce, false, "Replace node without confirmation prompt")
	AddBoolFlag(cmdKubeNodeReplace, "skip-drain", "", false, "Skip draining the node before replacement")

	return cmd
}

func kubernetesRegistryIntegration() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "registry",
			Aliases: []string{"reg"},
			Short:   "Display commands for integrating clusters with docr",
			Long:    "The commands under `registry` are for managing DOCR integration with Kubernetes clusters. You can use these commands to add or remove registry from one or more clusters.",
		},
	}

	k8sCmdService := kubernetesCommandService()

	CmdBuilder(cmd, k8sCmdService.RunKubernetesRegistryAdd,
		"add <cluster-id|cluster-name> <cluster-id|cluster-name>", "Add container registry support to Kubernetes clusters", `
This command adds container registry support to the specified Kubernetes cluster(s).`,
		Writer, aliasOpt("a"))

	CmdBuilder(cmd, k8sCmdService.RunKubernetesRegistryRemove,
		"remove <cluster-id|cluster-name> <cluster-id|cluster-name>", "Remove container registry support from Kubernetes clusters", `
This command removes container registry support from the specified Kubernetes cluster(s).`,
		Writer, aliasOpt("rm"))

	return cmd
}

// kubernetesOneClicks creates the 1-click command.
func kubernetesOneClicks() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "1-click",
			Short: "Display commands that pertain to kubernetes 1-click applications",
			Long:  "The commands under `doctl kubernetes 1-click` are for interacting with DigitalOcean Kubernetes 1-Click applications.",
		},
	}

	CmdBuilder(cmd, RunKubernetesOneClickList, "list", "Retrieve a list of Kubernetes 1-Click applications", "Use this command to retrieve a list of Kubernetes 1-Click applications.", Writer,
		aliasOpt("ls"), displayerType(&displayers.OneClick{}))
	cmdKubeOneClickInstall := CmdBuilder(cmd, RunKubernetesOneClickInstall, "install <cluster-id>", "Install 1-click apps on a Kubernetes cluster", "Use this command to install 1-click apps on a Kubernetes cluster using the flag --1-clicks.", Writer, aliasOpt("in"), displayerType(&displayers.OneClick{}))
	AddStringSliceFlag(cmdKubeOneClickInstall, doctl.ArgOneClicks, "", nil, "1-clicks to be installed on a Kubernetes cluster. Multiple 1-clicks can be added at once. Example: --1-clicks moon,loki,netdata")
	return cmd
}

// RunKubernetesOneClickList retrieves a list of 1-clicks for kubernetes.
func RunKubernetesOneClickList(c *CmdConfig) error {
	oneClicks := c.OneClicks()
	oneClickList, err := oneClicks.List("kubernetes")
	if err != nil {
		return err
	}

	items := &displayers.OneClick{OneClicks: oneClickList}

	return c.Display(items)
}

// RunKubernetesOneClickInstall installs 1-click apps on a kubernetes cluster.
func RunKubernetesOneClickInstall(c *CmdConfig) error {
	oneClicks := c.OneClicks()
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	oneClickSlice, err := c.Doit.GetStringSlice(c.NS, doctl.ArgOneClicks)
	if err != nil {
		return err
	}

	oneClickInstall, err := oneClicks.InstallKubernetes(c.Args[0], oneClickSlice)
	if err != nil {
		return err
	}

	notice(oneClickInstall)
	return nil
}

func kubernetesOptions() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "options",
			Aliases: []string{"opts", "o"},
			Short:   "List possible option values for use inside Kubernetes commands",
			Long:    "The `options` commands are used to enumerate values for use with `doctl`'s Kubernetes commands. This is useful in certain cases where flags only accept input that is from a list of possible values, such as Kubernetes versions, datacenter regions, and machine sizes.",
		},
	}

	k8sCmdService := kubernetesCommandService()

	k8sVersionDesc := "List Kubernetes versions that can be used with DigitalOcean clusters"
	CmdBuilder(cmd, k8sCmdService.RunKubeOptionsListVersion, "versions",
		k8sVersionDesc, k8sVersionDesc, Writer, aliasOpt("v"))
	k8sRegionsDesc := "List regions that support DigitalOcean Kubernetes clusters"
	CmdBuilder(cmd, k8sCmdService.RunKubeOptionsListRegion, "regions",
		k8sRegionsDesc, k8sRegionsDesc, Writer, aliasOpt("r"))
	k8sSizesDesc := "List machine sizes that can be used in a DigitalOcean Kubernetes cluster"
	CmdBuilder(cmd, k8sCmdService.RunKubeOptionsListNodeSizes, "sizes",
		k8sSizesDesc, k8sSizesDesc, Writer, aliasOpt("s"))
	return cmd
}

// Clusters

// RunKubernetesClusterGet retrieves an existing kubernetes cluster by its identifier.
func (s *KubernetesCommandService) RunKubernetesClusterGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	clusterIDorName := c.Args[0]

	cluster, err := clusterByIDorName(c.Kubernetes(), clusterIDorName)
	if err != nil {
		return err
	}
	return displayClusters(c, false, *cluster)
}

// RunKubernetesClusterList lists kubernetes.
func (s *KubernetesCommandService) RunKubernetesClusterList(c *CmdConfig) error {
	kube := c.Kubernetes()
	list, err := kube.List()
	if err != nil {
		return err
	}

	return displayClusters(c, true, list...)
}

// RunKubernetesClusterGetUpgrades retrieves available upgrade versions for a cluster.
func (s *KubernetesCommandService) RunKubernetesClusterGetUpgrades(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	clusterIDorName := c.Args[0]
	clusterID, err := clusterIDize(c, clusterIDorName)
	if err != nil {
		return err
	}

	kube := c.Kubernetes()

	upgrades, err := kube.GetUpgrades(clusterID)
	if err != nil {
		return err
	}

	item := &displayers.KubernetesVersions{KubernetesVersions: upgrades}
	return c.Display(item)
}

// RunKubernetesClusterCreate creates a new kubernetes with a given configuration.
func (s *KubernetesCommandService) RunKubernetesClusterCreate(defaultNodeSize string, defaultNodeCount int) func(*CmdConfig) error {
	return func(c *CmdConfig) error {
		err := ensureOneArg(c)
		if err != nil {
			return err
		}
		clusterName := c.Args[0]
		r := &godo.KubernetesClusterCreateRequest{Name: clusterName}
		if err := buildClusterCreateRequestFromArgs(c, r, defaultNodeSize, defaultNodeCount); err != nil {
			return err
		}
		wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
		if err != nil {
			return err
		}
		update, err := c.Doit.GetBool(c.NS, doctl.ArgClusterUpdateKubeconfig)
		if err != nil {
			return err
		}
		setCurrentContext, err := c.Doit.GetBool(c.NS, doctl.ArgSetCurrentContext)
		if err != nil {
			return err
		}

		kube := c.Kubernetes()

		cluster, err := kube.Create(r)
		if err != nil {
			return err
		}

		if wait {
			notice("Cluster is provisioning, waiting for cluster to be running")
			cluster, err = waitForClusterRunning(kube, cluster.ID)
			if err != nil {
				warn("Cluster couldn't enter `running` state: %v", err)
			}
		}

		if update {
			notice("Cluster created, fetching credentials")
			s.tryUpdateKubeconfig(kube, cluster.ID, clusterName, setCurrentContext)
		}

		oneClickApps, err := c.Doit.GetStringSlice(c.NS, doctl.ArgOneClicks)
		if err != nil {
			return err
		}
		if len(oneClickApps) > 0 {
			oneClicks := c.OneClicks()
			messageResponse, err := oneClicks.InstallKubernetes(cluster.ID, oneClickApps)
			if err != nil {
				warn("Failed to kick off 1-Click Application(s) install")
			} else {
				notice(messageResponse)
			}
		}

		return displayClusters(c, true, *cluster)
	}
}

// RunKubernetesClusterUpdate updates an existing kubernetes with new configuration.
func (s *KubernetesCommandService) RunKubernetesClusterUpdate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	update, err := c.Doit.GetBool(c.NS, doctl.ArgClusterUpdateKubeconfig)
	if err != nil {
		return err
	}
	setCurrentContext, err := c.Doit.GetBool(c.NS, doctl.ArgSetCurrentContext)
	if err != nil {
		return err
	}
	clusterIDorName := c.Args[0]
	clusterID, err := clusterIDize(c, clusterIDorName)
	if err != nil {
		return err
	}

	r := new(godo.KubernetesClusterUpdateRequest)
	if err := buildClusterUpdateRequestFromArgs(c, r); err != nil {
		return err
	}

	kube := c.Kubernetes()
	cluster, err := kube.Update(clusterID, r)
	if err != nil {
		return err
	}

	if update {
		notice("Cluster updated, fetching new credentials")
		s.tryUpdateKubeconfig(kube, clusterID, clusterIDorName, setCurrentContext)
	}

	return displayClusters(c, true, *cluster)
}

func (s *KubernetesCommandService) tryUpdateKubeconfig(kube do.KubernetesService, clusterID, clusterName string, setCurrentContext bool) {
	var (
		remoteConfig *clientcmdapi.Config
		err          error
	)
	ctx, cancel := context.WithTimeout(context.TODO(), timeoutFetchingKubeconfig)
	defer cancel()
	for {
		remoteConfig, err = s.KubeconfigProvider.Remote(kube, clusterID, 0)
		if err != nil {
			select {
			case <-ctx.Done():
				warn("Couldn't get credentials for cluster. It will not be added to your kubeconfig: %v", err)
				return
			case <-time.After(time.Second):
			}
		} else {
			break
		}
	}
	if err := s.writeOrAddToKubeconfig(clusterID, remoteConfig, setCurrentContext, 0); err != nil {
		warn("Couldn't write cluster credentials: %v", err)
	}
}

// RunKubernetesClusterUpgrade upgrades an existing cluster to a new version.
func (s *KubernetesCommandService) RunKubernetesClusterUpgrade(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}

	version, available, err := getUpgradeVersionOrLatest(c, clusterID)
	if err != nil {
		return err
	}
	if !available {
		notice("Cluster is already up-to-date; no upgrades available.")
		return nil
	}

	kube := c.Kubernetes()
	err = kube.Upgrade(clusterID, version)
	if err != nil {
		return err
	}

	notice("Upgrading cluster to version %v", version)
	return nil
}

func getUpgradeVersionOrLatest(c *CmdConfig, clusterID string) (string, bool, error) {
	version, err := c.Doit.GetString(c.NS, doctl.ArgClusterVersionSlug)
	if err != nil {
		return "", false, err
	}
	if version != "" && version != defaultKubernetesLatestVersion {
		return version, true, nil
	}

	cluster, err := c.Kubernetes().Get(clusterID)
	if err != nil {
		return "", false, fmt.Errorf("Unable to look up cluster to find the latest version from the API: %v", err)
	}

	versions, err := c.Kubernetes().GetUpgrades(clusterID)
	if err != nil {
		return "", false, fmt.Errorf("Unable to look up the latest version from the API: %v", err)
	}
	if len(versions) == 0 {
		return "", false, nil
	}

	return latestVersionForUpgrade(cluster.VersionSlug, versions)
}

// latestVersionForUpgrade returns the newest patch version from `versions` for
// the minor version of `clusterVersionSlug`. This ensures we never use a
// different minor version than a cluster is running as "latest" for an upgrade,
// since we want minor version upgrades to be an explicit operation.
func latestVersionForUpgrade(clusterVersionSlug string, versions []do.KubernetesVersion) (string, bool, error) {
	clusterSV, err := semver.Parse(clusterVersionSlug)
	if err != nil {
		return "", false, err
	}
	clusterBucket := fmt.Sprintf("%d.%d", clusterSV.Major, clusterSV.Minor)

	// Sort releases into minor-version buckets.
	var serr error
	releases := versionMapBy(versions, func(v do.KubernetesVersion) string {
		sv, err := semver.Parse(v.Slug)
		if err != nil {
			serr = err
			return ""
		}
		return fmt.Sprintf("%d.%d", sv.Major, sv.Minor)
	})
	if serr != nil {
		return "", false, serr
	}

	// Find the cluster's minor version in the bucketized available versions.
	bucket, ok := releases[clusterBucket]
	if !ok {
		// No upgrades available within the cluster's minor version.
		return "", false, nil
	}

	// Find the latest version within the bucket.
	i, err := versionMaxBy(bucket, func(v do.KubernetesVersion) string {
		return v.Slug
	})
	if err != nil {
		return "", false, err
	}

	return bucket[i].Slug, true, nil
}

// RunKubernetesClusterDelete deletes a Kubernetes cluster
func (s *KubernetesCommandService) RunKubernetesClusterDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	update, err := c.Doit.GetBool(c.NS, doctl.ArgClusterUpdateKubeconfig)
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	dangerous, err := c.Doit.GetBool(c.NS, doctl.ArgDangerous)
	if err != nil {
		return err
	}

	kube := c.Kubernetes()

	for _, cluster := range c.Args {
		clusterID, err := clusterIDize(c, cluster)
		if err != nil {
			return err
		}

		if force || AskForConfirmDelete("Kubernetes cluster", 1) == nil {
			// continue
		} else {
			return fmt.Errorf("Operation aborted")
		}

		var kubeconfig []byte
		if update {
			// get the cluster's kubeconfig before issuing the delete, so that we can
			// cleanup the entry from the local file
			kubeconfig, err = kube.GetKubeConfig(clusterID)
			if err != nil {
				warn("Couldn't get credentials for cluster. It will not be remove from your kubeconfig.")
			}
		}

		if dangerous {
			err = kube.DeleteDangerous(clusterID)
		} else {
			err = kube.Delete(clusterID)
		}
		if err != nil {
			return err
		}

		if kubeconfig != nil {
			notice("Cluster deleted, removing credentials")
			if err := removeFromKubeconfig(kubeconfig); err != nil {
				warn("Cluster was deleted, but we couldn't remove it from your local kubeconfig. Try doing it manually.")
			}
		}
	}

	return nil
}

func (s *KubernetesCommandService) RunKubernetesClusterDeleteSelective(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	clusterIDorName := c.Args[0]

	clusterID, err := clusterIDize(c, clusterIDorName)
	if err != nil {
		return err
	}

	update, err := c.Doit.GetBool(c.NS, doctl.ArgClusterUpdateKubeconfig)
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	volumes, err := c.Doit.GetStringSlice(c.NS, doctl.ArgVolumeList)
	if err != nil {
		return err
	}

	volSnapshots, err := c.Doit.GetStringSlice(c.NS, doctl.ArgVolumeSnapshotList)
	if err != nil {
		return err
	}

	loadBalancers, err := c.Doit.GetStringSlice(c.NS, doctl.ArgLoadBalancerList)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("Kubernetes cluster", 1) == nil {
		// continue
	} else {
		return fmt.Errorf("Operation aborted")
	}

	kube := c.Kubernetes()

	var kubeconfig []byte
	if update {
		// get the cluster's kubeconfig before issuing the delete, so that we can
		// cleanup the entry from the local file
		kubeconfig, err = kube.GetKubeConfig(clusterID)
		if err != nil {
			warn("Couldn't get credentials for cluster. It will not be remove from your kubeconfig.")
		}
	}

	cluster, err := kube.Get(clusterID)
	if err != nil {
		return err
	}

	volIDs := make([]string, 0, len(volumes))
	for _, v := range volumes {
		volumeID, err := iDize(c, v, "volume", cluster.RegionSlug)
		if err != nil {
			return err
		}
		volIDs = append(volIDs, volumeID)
	}

	snapshotIDs := make([]string, 0, len(volSnapshots))
	for _, s := range volSnapshots {
		snapID, err := iDize(c, s, "volume_snapshot", cluster.RegionSlug)
		if err != nil {
			return err
		}
		snapshotIDs = append(snapshotIDs, snapID)
	}

	lbIDs := make([]string, 0, len(loadBalancers))
	for _, l := range loadBalancers {
		lbID, err := iDize(c, l, "load_balancer", "")
		if err != nil {
			return err
		}
		lbIDs = append(lbIDs, lbID)
	}

	r := new(godo.KubernetesClusterDeleteSelectiveRequest)
	r.Volumes = volIDs
	r.VolumeSnapshots = snapshotIDs
	r.LoadBalancers = lbIDs

	err = kube.DeleteSelective(clusterID, r)
	if err != nil {
		return err
	}

	if kubeconfig != nil {
		notice("Cluster deleted, removing credentials")
		if err := removeFromKubeconfig(kubeconfig); err != nil {
			warn("Cluster was deleted, but we couldn't remove it from your local kubeconfig. Try doing it manually.")
		}
	}
	return nil
}

// RunKubernetesClusterListAssociatedResources lists a Kubernetes cluster's associated resources
func (s *KubernetesCommandService) RunKubernetesClusterListAssociatedResources(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	clusterIDorName := c.Args[0]

	clusterID, err := clusterIDize(c, clusterIDorName)
	if err != nil {
		return err
	}

	kube := c.Kubernetes()
	resources, err := kube.ListAssociatedResourcesForDeletion(clusterID)
	if err != nil {
		return err
	}

	return displayAssociatedResources(c, resources)
}

// Kubeconfig

// RunKubernetesKubeconfigShow retrieves an existing kubernetes config and prints it.
func (s *KubernetesCommandService) RunKubernetesKubeconfigShow(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	expirySeconds, err := c.Doit.GetInt(c.NS, doctl.ArgKubeConfigExpirySeconds)
	if err != nil {
		return err
	}

	kube := c.Kubernetes()
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}

	var kubeconfig []byte
	if expirySeconds > 0 {
		kubeconfig, err = kube.GetKubeConfigWithExpiry(clusterID, int64(expirySeconds))
	} else {
		kubeconfig, err = kube.GetKubeConfig(clusterID)
	}
	if err != nil {
		return err
	}

	_, err = c.Out.Write(kubeconfig)
	return err
}

func cachedExecCredentialPath(id string) string {
	return filepath.Join(kubeconfigCachePath(), id+".json")
}

// loadCachedExecCredential attempts to load the cached exec credential from disk. Never errors
// Returns nil if there's no credential, if it failed to load it, or if it's expired.
func loadCachedExecCredential(id string) (*clientauthentication.ExecCredential, error) {
	path := cachedExecCredentialPath(id)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	defer f.Close()

	var execCredential *clientauthentication.ExecCredential
	if err := json.NewDecoder(f).Decode(&execCredential); err != nil {
		return nil, err
	}

	if execCredential.Status == nil {
		// Invalid ExecCredential, remove it
		err = os.Remove(path)
		return nil, err
	}

	t := execCredential.Status.ExpirationTimestamp
	if t.IsZero() || t.Time.Before(time.Now()) {
		err = os.Remove(path)
		return nil, err
	}

	return execCredential, nil
}

// cacheExecCredential caches an ExecCredential to the doctl cache directory
func cacheExecCredential(id string, execCredential *clientauthentication.ExecCredential) error {
	// Don't bother caching if there's no expiration set
	if execCredential.Status.ExpirationTimestamp.IsZero() {
		return nil
	}

	cachePath := kubeconfigCachePath()
	if err := os.MkdirAll(cachePath, os.FileMode(0700)); err != nil {
		return err
	}

	path := cachedExecCredentialPath(id)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(0600))
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(execCredential)
}

// RunKubernetesKubeconfigExecCredential displays the exec credential. It is for internal use only.
func (s *KubernetesCommandService) RunKubernetesKubeconfigExecCredential(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	version, err := c.Doit.GetString(c.NS, doctl.ArgVersion)
	if err != nil {
		return err
	}

	if version != "v1beta1" {
		return fmt.Errorf("Invalid version %q, expected 'v1beta1'", version)
	}

	kube := c.Kubernetes()

	clusterID := c.Args[0]
	execCredential, err := loadCachedExecCredential(clusterID)
	if err != nil && Verbose {
		warn("%v", err)
	}

	if execCredential != nil {
		return json.NewEncoder(c.Out).Encode(execCredential)
	}

	credentials, err := kube.GetCredentials(clusterID)
	if err != nil {
		if errResponse, ok := err.(*godo.ErrorResponse); ok {
			return fmt.Errorf("Failed to fetch credentials for cluster %q: %v", clusterID, errResponse.Message)
		}
		return err
	}

	status := &clientauthentication.ExecCredentialStatus{
		ClientCertificateData: string(credentials.ClientCertificateData),
		ClientKeyData:         string(credentials.ClientKeyData),
		ExpirationTimestamp:   &metav1.Time{Time: credentials.ExpiresAt},
		Token:                 credentials.Token,
	}

	execCredential = &clientauthentication.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			Kind:       execCredentialKind,
			APIVersion: clientauthentication.SchemeGroupVersion.String(),
		},
		Status: status,
	}

	// Don't error out when caching credentials, just print it if we're being verbose
	if err := cacheExecCredential(clusterID, execCredential); err != nil && Verbose {
		warn("%v", err)
	}

	return json.NewEncoder(c.Out).Encode(execCredential)
}

// RunKubernetesKubeconfigSave retrieves an existing kubernetes config and saves it to your local kubeconfig.
func (s *KubernetesCommandService) RunKubernetesKubeconfigSave(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	expirySeconds, err := c.Doit.GetInt(c.NS, doctl.ArgKubeConfigExpirySeconds)
	if err != nil {
		return err
	}

	kube := c.Kubernetes()
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}

	remoteKubeconfig, err := s.KubeconfigProvider.Remote(kube, clusterID, expirySeconds)
	if err != nil {
		return err
	}

	alias, err := c.Doit.GetString(c.NS, doctl.ArgKubernetesAlias)
	if err != nil {
		return err
	}

	if alias != "" {
		remoteKubeconfig.Contexts[alias] = remoteKubeconfig.Contexts[remoteKubeconfig.CurrentContext]
		delete(remoteKubeconfig.Contexts, remoteKubeconfig.CurrentContext)
		remoteKubeconfig.CurrentContext = alias
	}

	setCurrentContext, err := c.Doit.GetBool(c.NS, doctl.ArgSetCurrentContext)
	if err != nil {
		return err
	}

	path := cachedExecCredentialPath(clusterID)
	_, err = os.Stat(path)
	if err == nil {
		os.Remove(path)
	}

	return s.writeOrAddToKubeconfig(clusterID, remoteKubeconfig, setCurrentContext, expirySeconds)
}

// RunKubernetesKubeconfigRemove retrieves an existing kubernetes config and removes it from your local kubeconfig.
func (s *KubernetesCommandService) RunKubernetesKubeconfigRemove(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	kube := c.Kubernetes()
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}
	kubeconfig, err := kube.GetKubeConfig(clusterID)
	if err != nil {
		return err
	}

	return removeFromKubeconfig(kubeconfig)
}

// Node Pools

// RunKubernetesNodePoolGet retrieves an existing cluster node pool by its identifier.
func (s *KubernetesCommandService) RunKubernetesNodePoolGet(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}
	nodePool, err := poolByIDorName(c.Kubernetes(), clusterID, c.Args[1])
	if err != nil {
		return err
	}
	return displayNodePools(c, *nodePool)
}

// RunKubernetesNodePoolList lists cluster node pool.
func (s *KubernetesCommandService) RunKubernetesNodePoolList(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}
	kube := c.Kubernetes()
	list, err := kube.ListNodePools(clusterID)
	if err != nil {
		return err
	}

	return displayNodePools(c, list...)
}

// RunKubernetesNodePoolCreate creates a new cluster node pool with a given configuration.
func (s *KubernetesCommandService) RunKubernetesNodePoolCreate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}

	r := new(godo.KubernetesNodePoolCreateRequest)
	if err := buildNodePoolCreateRequestFromArgs(c, r); err != nil {
		return err
	}

	kube := c.Kubernetes()
	nodePool, err := kube.CreateNodePool(clusterID, r)
	if err != nil {
		return err
	}

	return displayNodePools(c, *nodePool)
}

// RunKubernetesNodePoolUpdate updates an existing cluster node pool with new properties.
func (s *KubernetesCommandService) RunKubernetesNodePoolUpdate(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}
	poolID, err := poolIDize(c.Kubernetes(), clusterID, c.Args[1])
	if err != nil {
		return err
	}

	r := new(godo.KubernetesNodePoolUpdateRequest)
	if err := buildNodePoolUpdateRequestFromArgs(c, r); err != nil {
		return err
	}

	kube := c.Kubernetes()
	nodePool, err := kube.UpdateNodePool(clusterID, poolID, r)
	if err != nil {
		return err
	}

	return displayNodePools(c, *nodePool)
}

// RunKubernetesNodePoolRecycle DEPRECATED: will be removed in v2.0, please use delete-node or replace-node
func (s *KubernetesCommandService) RunKubernetesNodePoolRecycle(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}
	poolID, err := poolIDize(c.Kubernetes(), clusterID, c.Args[1])
	if err != nil {
		return err
	}

	r := new(godo.KubernetesNodePoolRecycleNodesRequest)
	if err := buildNodePoolRecycleRequestFromArgs(c, clusterID, poolID, r); err != nil {
		return err
	}

	kube := c.Kubernetes()
	return kube.RecycleNodePoolNodes(clusterID, poolID, r)
}

// RunKubernetesNodePoolDelete deletes a Kubernetes node pool
func (s *KubernetesCommandService) RunKubernetesNodePoolDelete(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}
	poolID, err := poolIDize(c.Kubernetes(), clusterID, c.Args[1])
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if force || AskForConfirmDelete("Kubernetes node pool", 1) == nil {
		kube := c.Kubernetes()
		if err := kube.DeleteNodePool(clusterID, poolID); err != nil {
			return err
		}
	} else {
		return errOperationAborted
	}
	return nil
}

// RunKubernetesNodeDelete deletes a Kubernetes Node
func (s *KubernetesCommandService) RunKubernetesNodeDelete(c *CmdConfig) error {
	return kubernetesNodeDelete(false, c)
}

// RunKubernetesNodeReplace replaces a Kubernetes Node
func (s *KubernetesCommandService) RunKubernetesNodeReplace(c *CmdConfig) error {
	return kubernetesNodeDelete(true, c)
}

func kubernetesNodeDelete(replace bool, c *CmdConfig) error {
	if len(c.Args) != 3 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c, c.Args[0])
	if err != nil {
		return err
	}
	poolID, err := poolIDize(c.Kubernetes(), clusterID, c.Args[1])
	if err != nil {
		return err
	}
	nodeID := c.Args[2]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	msg := "delete this Kubernetes node?"
	if replace {
		msg = "replace this Kubernetes node?"
	}

	if !(force || AskForConfirm(msg) == nil) {
		return errOperationAborted
	}

	skipDrain, err := c.Doit.GetBool(c.NS, "skip-drain")
	if err != nil {
		return err
	}

	kube := c.Kubernetes()
	return kube.DeleteNode(clusterID, poolID, nodeID, &godo.KubernetesNodeDeleteRequest{
		Replace:   replace,
		SkipDrain: skipDrain,
	})
}

// RunKubeOptionsListVersion lists valid versions for kubernetes clusters.
func (s *KubernetesCommandService) RunKubeOptionsListVersion(c *CmdConfig) error {
	kube := c.Kubernetes()
	versions, err := kube.GetVersions()
	if err != nil {
		return err
	}
	item := &displayers.KubernetesVersions{KubernetesVersions: versions}
	return c.Display(item)
}

// RunKubeOptionsListRegion lists valid regions for kubernetes clusters.
func (s *KubernetesCommandService) RunKubeOptionsListRegion(c *CmdConfig) error {
	kube := c.Kubernetes()
	regions, err := kube.GetRegions()
	if err != nil {
		return err
	}
	item := &displayers.KubernetesRegions{KubernetesRegions: regions}
	return c.Display(item)
}

// RunKubeOptionsListNodeSizes lists valid node sizes for kubernetes clusters.
func (s *KubernetesCommandService) RunKubeOptionsListNodeSizes(c *CmdConfig) error {
	kube := c.Kubernetes()
	sizes, err := kube.GetNodeSizes()
	if err != nil {
		return err
	}
	item := &displayers.KubernetesNodeSizes{KubernetesNodeSizes: sizes}
	return c.Display(item)
}

func (s *KubernetesCommandService) RunKubernetesRegistryAdd(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterUUIDs := make([]string, 0, len(c.Args))
	for _, arg := range c.Args {
		clusterID, err := clusterIDize(c, arg)
		if err != nil {
			return err
		}
		clusterUUIDs = append(clusterUUIDs, clusterID)
	}
	r := new(godo.KubernetesClusterRegistryRequest)
	r.ClusterUUIDs = clusterUUIDs

	kube := c.Kubernetes()
	return kube.AddRegistry(r)
}

func (s *KubernetesCommandService) RunKubernetesRegistryRemove(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterUUIDs := make([]string, 0, len(c.Args))
	for _, arg := range c.Args {
		clusterID, err := clusterIDize(c, arg)
		if err != nil {
			return err
		}
		clusterUUIDs = append(clusterUUIDs, clusterID)
	}
	r := new(godo.KubernetesClusterRegistryRequest)
	r.ClusterUUIDs = clusterUUIDs

	kube := c.Kubernetes()
	return kube.RemoveRegistry(r)
}

func buildClusterCreateRequestFromArgs(c *CmdConfig, r *godo.KubernetesClusterCreateRequest, defaultNodeSize string, defaultNodeCount int) error {
	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	r.RegionSlug = region

	vpcUUID, err := c.Doit.GetString(c.NS, doctl.ArgClusterVPCUUID)
	if err != nil {
		return err
	}
	// empty "" is fine, the default region VPC will be resolved
	r.VPCUUID = vpcUUID

	version, err := getVersionOrLatest(c)
	if err != nil {
		return err
	}
	r.VersionSlug = version

	autoUpgrade, err := c.Doit.GetBool(c.NS, doctl.ArgAutoUpgrade)
	if err != nil {
		return err
	}
	r.AutoUpgrade = autoUpgrade

	surgeUpgrade, err := c.Doit.GetBool(c.NS, doctl.ArgSurgeUpgrade)
	if err != nil {
		return err
	}
	r.SurgeUpgrade = surgeUpgrade

	ha, err := c.Doit.GetBool(c.NS, doctl.ArgHA)
	if err != nil {
		return err
	}
	r.HA = ha

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return err
	}
	r.Tags = tags

	maintenancePolicy, err := parseMaintenancePolicy(c)
	if err != nil {
		return err
	}
	r.MaintenancePolicy = maintenancePolicy

	// node pools
	nodePoolSpecs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgClusterNodePool)
	if err != nil {
		return err
	}

	if len(nodePoolSpecs) == 0 {
		nodePoolSize, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
		if err != nil {
			return err
		}

		nodePoolCount, err := c.Doit.GetInt(c.NS, doctl.ArgNodePoolCount)
		if err != nil {
			return err
		}

		nodePoolName := r.Name + "-default-pool"
		r.NodePools = []*godo.KubernetesNodePoolCreateRequest{{
			Name:  nodePoolName,
			Size:  nodePoolSize,
			Count: nodePoolCount,
		}}

		return nil
	}

	// multiple node pools
	if c.Doit.IsSet(doctl.ArgSizeSlug) || c.Doit.IsSet(doctl.ArgNodePoolCount) {
		return fmt.Errorf("Flags %q and %q cannot be provided when %q is present", doctl.ArgSizeSlug, doctl.ArgNodePoolCount, doctl.ArgClusterNodePool)
	}

	nodePools, err := buildNodePoolCreateRequestsFromArgs(c, nodePoolSpecs, r.Name, defaultNodeSize, defaultNodeCount)
	if err != nil {
		return err
	}
	r.NodePools = nodePools

	return nil
}

func buildClusterUpdateRequestFromArgs(c *CmdConfig, r *godo.KubernetesClusterUpdateRequest) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgClusterName)
	if err != nil {
		return err
	}
	r.Name = name

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return err
	}
	r.Tags = tags

	maintenancePolicy, err := parseMaintenancePolicy(c)
	if err != nil {
		return err
	}
	r.MaintenancePolicy = maintenancePolicy

	autoUpgrade, err := c.Doit.GetBoolPtr(c.NS, doctl.ArgAutoUpgrade)
	if err != nil {
		return err
	}
	r.AutoUpgrade = autoUpgrade

	surgeUpgrade, err := c.Doit.GetBool(c.NS, doctl.ArgSurgeUpgrade)
	if err != nil {
		return err
	}
	r.SurgeUpgrade = surgeUpgrade

	return nil
}

func buildNodePoolRecycleRequestFromArgs(c *CmdConfig, clusterID, poolID string, r *godo.KubernetesNodePoolRecycleNodesRequest) error {
	nodeIDorNames, err := c.Doit.GetStringSlice(c.NS, doctl.ArgNodePoolNodeIDs)
	if err != nil {
		return err
	}
	allUUIDs := true
	for _, node := range nodeIDorNames {
		if !looksLikeUUID(node) {
			allUUIDs = false
		}
	}
	if allUUIDs {
		r.Nodes = nodeIDorNames
	} else {
		// at least some args weren't UUIDs, so assume that they're all names
		nodes, err := nodesByNames(c.Kubernetes(), clusterID, poolID, nodeIDorNames)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			r.Nodes = append(r.Nodes, node.ID)
		}
	}
	return nil
}

func buildNodePoolCreateRequestsFromArgs(c *CmdConfig, nodePools []string, clusterName, defaultSize string, defaultCount int) ([]*godo.KubernetesNodePoolCreateRequest, error) {
	out := make([]*godo.KubernetesNodePoolCreateRequest, 0, len(nodePools))
	for i, nodePoolString := range nodePools {
		defaultName := fmt.Sprintf("%s-pool-%d", clusterName, i+1)
		poolCreateReq, err := parseNodePoolString(nodePoolString, defaultName, defaultSize, defaultCount)
		if err != nil {
			return nil, fmt.Errorf("Invalid node pool arguments for flag %d: %v", i, err)
		}
		out = append(out, poolCreateReq)
	}
	return out, nil
}

func parseNodePoolString(nodePool, defaultName, defaultSize string, defaultCount int) (*godo.KubernetesNodePoolCreateRequest, error) {
	const (
		argSeparator = ";"
		kvSeparator  = "="
	)
	out := &godo.KubernetesNodePoolCreateRequest{
		Name:   defaultName,
		Size:   defaultSize,
		Count:  defaultCount,
		Labels: map[string]string{},
		Taints: []godo.Taint{},
	}
	trimmedPool := strings.TrimSuffix(nodePool, argSeparator)
	for _, arg := range strings.Split(trimmedPool, argSeparator) {
		kvs := strings.SplitN(arg, kvSeparator, 2)
		if len(kvs) < 2 {
			return nil, fmt.Errorf("A node pool string argument must be of the form `key=value`. Provided KVs: %v", kvs)
		}
		key := kvs[0]
		value := kvs[1]
		switch key {
		case "name":
			out.Name = value
		case "size":
			out.Size = value
		case "count":
			count, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, errors.New("Node pool count must be a valid integer.")
			}
			out.Count = int(count)
		case "tag":
			out.Tags = append(out.Tags, value)
		case "label":
			labelParts := strings.SplitN(value, "=", 2)
			if len(labelParts) < 2 {
				return nil, fmt.Errorf("a node pool label component must be of the form `label-key=label-value`, got %q", value)
			}
			labelKey := labelParts[0]
			labelValue := labelParts[1]
			out.Labels[labelKey] = labelValue
		case "taint":
			taint, err := parseTaint(value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse taint: %s", err)
			}
			out.Taints = append(out.Taints, taint)
		case "auto-scale":
			autoScale, err := strconv.ParseBool(value)
			if err != nil {
				return nil, errors.New("Node pool auto-scale value must be a valid boolean.")
			}
			out.AutoScale = autoScale
		case "min-nodes":
			minNodes, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, errors.New("Node pool min-nodes must be a valid integer.")
			}
			out.MinNodes = int(minNodes)
		case "max-nodes":
			maxNodes, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, errors.New("Node pool max-nodes must be a valid integer.")
			}
			out.MaxNodes = int(maxNodes)
		default:
			return nil, fmt.Errorf("Unsupported node pool argument %q", key)
		}
	}
	return out, nil
}

func buildNodePoolCreateRequestFromArgs(c *CmdConfig, r *godo.KubernetesNodePoolCreateRequest) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgNodePoolName)
	if err != nil {
		return err
	}
	r.Name = name

	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return err
	}
	r.Size = size

	count, err := c.Doit.GetIntPtr(c.NS, doctl.ArgNodePoolCount)
	if err != nil {
		return err
	}
	if count == nil {
		count = intPtr(0)
	}
	r.Count = *count

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return err
	}
	r.Tags = tags

	labels, err := c.Doit.GetStringMapString(c.NS, doctl.ArgKubernetesLabel)
	if err != nil {
		return err
	}
	r.Labels = labels

	rawTaints, err := c.Doit.GetStringSlice(c.NS, doctl.ArgKubernetesTaint)
	if err != nil {
		return err
	}
	taints, err := parseTaints(rawTaints)
	if err != nil {
		return fmt.Errorf("failed to parse taints: %s", err)
	}
	r.Taints = taints

	autoScale, err := c.Doit.GetBool(c.NS, doctl.ArgNodePoolAutoScale)
	if err != nil {
		return err
	}
	r.AutoScale = autoScale

	minNodes, err := c.Doit.GetInt(c.NS, doctl.ArgNodePoolMinNodes)
	if err != nil {
		return err
	}
	r.MinNodes = minNodes

	maxNodes, err := c.Doit.GetInt(c.NS, doctl.ArgNodePoolMaxNodes)
	if err != nil {
		return err
	}
	r.MaxNodes = maxNodes

	return nil
}

func buildNodePoolUpdateRequestFromArgs(c *CmdConfig, r *godo.KubernetesNodePoolUpdateRequest) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgNodePoolName)
	if err != nil {
		return err
	}
	r.Name = name

	count, err := c.Doit.GetIntPtr(c.NS, doctl.ArgNodePoolCount)
	if err != nil {
		return err
	}
	r.Count = count

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return err
	}
	r.Tags = tags

	labels, err := c.Doit.GetStringMapString(c.NS, doctl.ArgKubernetesLabel)
	if err != nil {
		return err
	}
	r.Labels = labels

	// Check if the taints flag is set so that we can distinguish between not
	// setting any taints, setting the empty taint (which equals clearing all
	// taints), and setting one or more non-empty taints.
	if c.Doit.IsSet(doctl.ArgKubernetesTaint) {
		rawTaints, err := c.Doit.GetStringSlice(c.NS, doctl.ArgKubernetesTaint)
		if err != nil {
			return err
		}
		taints, err := parseTaints(rawTaints)
		if err != nil {
			return fmt.Errorf("failed to parse taints: %s", err)
		}
		r.Taints = &taints
	}

	autoScale, err := c.Doit.GetBoolPtr(c.NS, doctl.ArgNodePoolAutoScale)
	if err != nil {
		return err
	}
	r.AutoScale = autoScale

	minNodes, err := c.Doit.GetIntPtr(c.NS, doctl.ArgNodePoolMinNodes)
	if err != nil {
		return err
	}
	r.MinNodes = minNodes

	maxNodes, err := c.Doit.GetIntPtr(c.NS, doctl.ArgNodePoolMaxNodes)
	if err != nil {
		return err
	}
	r.MaxNodes = maxNodes

	return nil
}

func (s *KubernetesCommandService) writeOrAddToKubeconfig(clusterID string, remoteKubeconfig *clientcmdapi.Config, setCurrentContext bool, expirySeconds int) error {
	localKubeconfig, err := s.KubeconfigProvider.Local()
	if err != nil {
		return err
	}

	kubectlDefaults := s.KubeconfigProvider.ConfigPath()
	notice("Adding cluster credentials to kubeconfig file found in %q", kubectlDefaults)
	if err := mergeKubeconfig(clusterID, remoteKubeconfig, localKubeconfig, setCurrentContext, expirySeconds); err != nil {
		return fmt.Errorf("Couldn't use the kubeconfig info received, %v", err)
	}

	return s.KubeconfigProvider.Write(localKubeconfig)
}

func removeFromKubeconfig(kubeconfig []byte) error {
	remote, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return err
	}
	kubectlDefaults := clientcmd.NewDefaultPathOptions()
	currentConfig, err := kubectlDefaults.GetStartingConfig()
	if err != nil {
		return err
	}
	notice("Removing cluster credentials from kubeconfig file found in %q", kubectlDefaults.GlobalFile)
	if err := removeKubeconfig(remote, currentConfig); err != nil {
		return fmt.Errorf("Couldn't use the kubeconfig info received, %v", err)
	}
	return clientcmd.ModifyConfig(kubectlDefaults, *currentConfig, false)
}

// mergeKubeconfig merges a remote cluster's config file with a local config file,
// assuming that the current context in the remote config file points to the
// cluster details to add to the local config.
func mergeKubeconfig(clusterID string, remote, local *clientcmdapi.Config, setCurrentContext bool, expirySeconds int) error {
	remoteCtx, ok := remote.Contexts[remote.CurrentContext]
	if !ok {
		// this is a bug in the backend, we received incomplete/non-sensical data
		return fmt.Errorf("The remote config has no context entry named %q. This is a bug, please open a ticket with DigitalOcean.",
			remote.CurrentContext,
		)
	}
	remoteCluster, ok := remote.Clusters[remoteCtx.Cluster]
	if !ok {
		// this is a bug in the backend, we received incomplete/non-sensical data
		return fmt.Errorf("The remote config has no cluster entry named %q. This is a bug, please open a ticket with DigitalOcean.",
			remoteCtx.Cluster,
		)
	}

	local.Contexts[remote.CurrentContext] = remoteCtx
	local.Clusters[remoteCtx.Cluster] = remoteCluster

	if setCurrentContext {
		notice("Setting current-context to %s", remote.CurrentContext)
		local.CurrentContext = remote.CurrentContext
	}

	switch {
	case expirySeconds > 0:
		// When expirySeconds is passed, token based auth should be used as
		// credentials should expire and not be renewed automatically
		local.AuthInfos[remoteCtx.AuthInfo] = &clientcmdapi.AuthInfo{
			Token: remote.AuthInfos[remoteCtx.AuthInfo].Token,
		}
	default:
		// Configure kubectl to call doctl to renew credentials automatically
		local.AuthInfos[remoteCtx.AuthInfo] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				APIVersion: clientauthentication.SchemeGroupVersion.String(),
				Command:    doctl.CommandName(),
				Args: []string{
					"kubernetes",
					"cluster",
					"kubeconfig",
					"exec-credential",
					"--version=v1beta1",
					"--context=" + getCurrentAuthContextFn(),
					clusterID,
				},
			},
		}
	}

	return nil
}

// removeKubeconfig removes a remote cluster's config file from a local config file,
// assuming that the current context in the remote config file points to the
// cluster details to remove from the local config.
func removeKubeconfig(remote, local *clientcmdapi.Config) error {
	remoteCtx, ok := remote.Contexts[remote.CurrentContext]
	if !ok {
		// this is a bug in the backend, we received incomplete/non-sensical data
		return fmt.Errorf("The remote config has no context entry named %q. This is a bug, please open a ticket with DigitalOcean.",
			remote.CurrentContext,
		)
	}

	delete(local.Contexts, remote.CurrentContext)
	delete(local.Clusters, remoteCtx.Cluster)
	delete(local.AuthInfos, remoteCtx.AuthInfo)
	if local.CurrentContext == remote.CurrentContext {
		local.CurrentContext = ""
		notice("The removed cluster was set as the current context in kubectl. Run `kubectl config get-contexts` to see a list of other contexts you can use, and `kubectl config set-context` to specify a new one.")
	}
	return nil
}

// waitForClusterRunning waits for a cluster to be running.
func waitForClusterRunning(kube do.KubernetesService, clusterID string) (*do.KubernetesCluster, error) {
	failCount := 0
	printNewLineSet := false
	for i := 0; ; i++ {
		if i != 0 {
			fmt.Fprint(os.Stderr, ".")
			if !printNewLineSet {
				printNewLineSet = true
				defer fmt.Fprintln(os.Stderr)
			}
		}
		cluster, err := kube.Get(clusterID)
		if err == nil {
			failCount = 0
		} else {
			// Allow for transient API failures
			failCount++
			if failCount >= maxAPIFailures {
				return nil, err
			}
		}

		if cluster == nil || cluster.Status == nil {
			time.Sleep(1 * time.Second)
			continue
		}
		switch cluster.Status.State {
		case godo.KubernetesClusterStatusRunning:
			return cluster, nil
		case godo.KubernetesClusterStatusProvisioning:
			time.Sleep(5 * time.Second)
		default:
			return cluster, fmt.Errorf("Unknown status: [%s]", cluster.Status.State)
		}
	}
}

func displayClusters(c *CmdConfig, short bool, clusters ...do.KubernetesCluster) error {
	item := &displayers.KubernetesClusters{KubernetesClusters: do.KubernetesClusters(clusters), Short: short}
	return c.Display(item)
}

func displayNodePools(c *CmdConfig, nodePools ...do.KubernetesNodePool) error {
	item := &displayers.KubernetesNodePools{KubernetesNodePools: do.KubernetesNodePools(nodePools)}
	return c.Display(item)
}

func displayAssociatedResources(c *CmdConfig, ar *do.KubernetesAssociatedResources) error {
	item := &displayers.KubernetesAssociatedResources{KubernetesAssociatedResources: ar}
	return c.Display(item)
}

// clusterByIDorName attempts to find a cluster by ID or by name if the argument isn't an ID. If multiple
// clusters have the same name, then an error with the cluster IDs matching this name is returned.
func clusterByIDorName(kube do.KubernetesService, idOrName string) (*do.KubernetesCluster, error) {
	if looksLikeUUID(idOrName) {
		clusterID := idOrName
		return kube.Get(clusterID)
	}
	clusters, err := kube.List()
	if err != nil {
		return nil, err
	}
	var out []*do.KubernetesCluster
	for _, c := range clusters {
		c1 := c
		if c.Name == idOrName {
			out = append(out, &c1)
		}
	}
	switch {
	case len(out) == 0:
		return nil, errNoClusterByName(idOrName)
	case len(out) > 1:
		var ids []string
		for _, c := range out {
			ids = append(ids, c.ID)
		}
		return nil, errAmbiguousClusterName(idOrName, ids)
	default:
		if len(out) != 1 {
			panic("The default case should always have len(out) == 1.")
		}
		return out[0], nil
	}
}

// clusterIDize attempts to make a cluster ID/name string be a cluster ID.
// use this as opposed to `clusterByIDorName` if you just care about getting
// a cluster ID and don't need the cluster object itself
func clusterIDize(c *CmdConfig, idOrName string) (string, error) {
	return iDize(c, idOrName, "cluster", "")
}

// iDize attempts to make a resource ID/name string be a resource ID.
// use this if you just care about getting a resource ID and don't need the object itself
func iDize(c *CmdConfig, resourceIDOrName string, resType string, regionSlug string) (string, error) {
	if looksLikeUUID(resourceIDOrName) {
		return resourceIDOrName, nil
	}
	var ids []string

	switch resType {
	case "volume":
		volumes, err := c.Volumes().List()
		if err != nil {
			return "", err
		}

		for _, v := range volumes {
			if v.Name == resourceIDOrName && v.Region.Slug == regionSlug {
				id := v.ID
				ids = append(ids, id)
			}
		}
	case "volume_snapshot":
		volSnapshots, err := c.Snapshots().ListVolume()
		if err != nil {
			return "", err
		}

		for _, v := range volSnapshots {
			if v.Name == resourceIDOrName && contains(v.Regions, regionSlug) {
				id := v.ID
				ids = append(ids, id)
			}
		}
	case "load_balancer":
		loadBalancers, err := c.LoadBalancers().List()
		if err != nil {
			return "", err
		}
		for _, l := range loadBalancers {
			if l.Name == resourceIDOrName {
				id := l.ID
				ids = append(ids, id)
			}
		}
	case "cluster":
		clusters, err := c.Kubernetes().List()
		if err != nil {
			return "", err
		}
		for _, c := range clusters {
			if c.Name == resourceIDOrName {
				id := c.ID
				ids = append(ids, id)
			}
		}
	}

	switch {
	case len(ids) == 0:
		return "", fmt.Errorf("no %s goes by the name %q", resType, resourceIDOrName)
	case len(ids) > 1:
		return "", fmt.Errorf("many %ss go by the name %q, they have the following IDs: %v", resType, resourceIDOrName, ids)
	default:
		if len(ids) != 1 {
			panic("The default case should always have len(ids) == 1.")
		}
		return ids[0], nil
	}
}

func contains(regions []string, region string) bool {
	for _, r := range regions {
		if r == region {
			return true
		}
	}
	return false
}

// poolByIDorName attempts to find a pool by ID or by name if the argument isn't an ID. If multiple
// pools have the same name, then an error with the pool IDs matching this name is returned.
func poolByIDorName(kube do.KubernetesService, clusterID, idOrName string) (*do.KubernetesNodePool, error) {
	if looksLikeUUID(idOrName) {
		poolID := idOrName
		return kube.GetNodePool(clusterID, poolID)
	}
	nodePools, err := kube.ListNodePools(clusterID)
	if err != nil {
		return nil, err
	}
	var out []*do.KubernetesNodePool
	for _, c := range nodePools {
		c1 := c
		if c.Name == idOrName {
			out = append(out, &c1)
		}
	}
	switch {
	case len(out) == 0:
		return nil, errNoPoolByName(idOrName)
	case len(out) > 1:
		var ids []string
		for _, c := range out {
			ids = append(ids, c.ID)
		}
		return nil, errAmbiguousPoolName(idOrName, ids)
	default:
		if len(out) != 1 {
			panic("The default case should always have len(out) == 1.")
		}
		return out[0], nil
	}
}

// poolIDize attempts to make a node pool ID/name string be a node pool ID.
// use this as opposed to `poolByIDorName` if you just care about getting
// a node pool ID and don't need the node pool object itself
func poolIDize(kube do.KubernetesService, clusterID, idOrName string) (string, error) {
	if looksLikeUUID(idOrName) {
		return idOrName, nil
	}
	pools, err := kube.ListNodePools(clusterID)
	if err != nil {
		return "", err
	}
	var ids []string
	for _, c := range pools {
		if c.Name == idOrName {
			ids = append(ids, c.ID)
		}
	}
	switch {
	case len(ids) == 0:
		return "", errNoPoolByName(idOrName)
	case len(ids) > 1:
		return "", errAmbiguousPoolName(idOrName, ids)
	default:
		if len(ids) != 1 {
			panic("The default case should always have len(ids) == 1.")
		}
		return ids[0], nil
	}
}

// nodesByNames attempts to find nodes by names. If multiple nodes have the same name,
// then an error with the node IDs matching this name is returned.
func nodesByNames(kube do.KubernetesService, clusterID, poolID string, nodeNames []string) ([]*godo.KubernetesNode, error) {
	nodePool, err := kube.GetNodePool(clusterID, poolID)
	if err != nil {
		return nil, err
	}
	out := make([]*godo.KubernetesNode, 0, len(nodeNames))
	for _, name := range nodeNames {
		node, err := nodeByName(name, nodePool.Nodes)
		if err != nil {
			return nil, err
		}
		out = append(out, node)
	}
	return out, nil
}

func nodeByName(name string, nodes []*godo.KubernetesNode) (*godo.KubernetesNode, error) {
	var out []*godo.KubernetesNode
	for _, n := range nodes {
		n1 := n
		if n.Name == name {
			out = append(out, n1)
		}
	}
	switch {
	case len(out) == 0:
		return nil, errNoClusterNodeByName(name)
	case len(out) > 1:
		var ids []string
		for _, c := range out {
			ids = append(ids, c.ID)
		}
		return nil, errAmbiguousClusterNodeName(name, ids)
	default:
		if len(out) != 1 {
			panic("The default case should always have len(out) == 1.")
		}
		return out[0], nil
	}
}

func looksLikeUUID(str string) bool {
	_, err := uuid.Parse(str)
	if err != nil {
		return false
	}

	// support only hyphenated UUIDs
	return strings.Contains(str, "-")
}

func getVersionOrLatest(c *CmdConfig) (string, error) {
	version, err := c.Doit.GetString(c.NS, doctl.ArgClusterVersionSlug)
	if err != nil {
		return "", err
	}
	if version != "" && version != defaultKubernetesLatestVersion {
		return version, nil
	}
	versions, err := c.Kubernetes().GetVersions()
	if err != nil {
		return "", fmt.Errorf("No version flag provided. Unable to lookup the latest version from the API: %v", err)
	}
	if len(versions) > 0 {
		return versions[0].Slug, nil
	}
	releases, err := latestReleases(versions)
	if err != nil {
		return "", err
	}
	i, err := versionMaxBy(releases, func(v do.KubernetesVersion) string {
		return v.KubernetesVersion.KubernetesVersion
	})
	if err != nil {
		return "", err
	}
	return releases[i].Slug, nil
}

func parseMaintenancePolicy(c *CmdConfig) (*godo.KubernetesMaintenancePolicy, error) {
	maintenanceWindow, err := c.Doit.GetString(c.NS, doctl.ArgMaintenanceWindow)
	if err != nil {
		return nil, err
	}

	splitted := strings.SplitN(maintenanceWindow, "=", 2)
	if len(splitted) != 2 {
		return nil, fmt.Errorf("A maintenance window argument must be of the form `day=HH:MM`, got: %v", splitted)
	}

	day, err := godo.KubernetesMaintenanceToDay(splitted[0])
	if err != nil {
		return nil, err
	}

	return &godo.KubernetesMaintenancePolicy{
		StartTime: splitted[1],
		Day:       day,
	}, nil
}

func latestReleases(versions []do.KubernetesVersion) ([]do.KubernetesVersion, error) {
	versionsByK8S := versionMapBy(versions, func(v do.KubernetesVersion) string {
		return v.KubernetesVersion.KubernetesVersion
	})

	out := make([]do.KubernetesVersion, 0, len(versionsByK8S))
	for _, versions := range versionsByK8S {
		i, err := versionMaxBy(versions, func(v do.KubernetesVersion) string {
			return v.Slug
		})
		if err != nil {
			return nil, err
		}
		out = append(out, versions[i])
	}
	var serr error
	out = versionSortBy(out, func(i, j do.KubernetesVersion) bool {
		iv, err := semver.Parse(i.KubernetesVersion.KubernetesVersion)
		if err != nil {
			serr = err
			return false
		}
		jv, err := semver.Parse(j.KubernetesVersion.KubernetesVersion)
		if err != nil {
			serr = err
			return false
		}
		return iv.LT(jv)
	})
	return out, serr
}

func versionMapBy(versions []do.KubernetesVersion, selector func(do.KubernetesVersion) string) map[string][]do.KubernetesVersion {
	m := make(map[string][]do.KubernetesVersion)
	for _, v := range versions {
		key := selector(v)
		m[key] = append(m[key], v)
	}
	return m
}

func versionMaxBy(versions []do.KubernetesVersion, selector func(do.KubernetesVersion) string) (int, error) {
	if len(versions) == 0 {
		return -1, nil
	}
	if len(versions) == 1 {
		return 0, nil
	}
	max := 0
	maxSV, err := semver.Parse(selector(versions[max]))
	if err != nil {
		return max, err
	}
	// NOTE: We have to iterate over all versions here even though we know
	// versions[0] won't be greater than maxSV so that the index i will be a
	// valid index into versions rather than into versions[1:].
	for i, v := range versions {
		sv, err := semver.Parse(selector(v))
		if err != nil {
			return max, err
		}
		if sv.GT(maxSV) {
			max = i
			maxSV = sv
		}
	}
	return max, nil
}

func versionSortBy(versions []do.KubernetesVersion, less func(i, j do.KubernetesVersion) bool) []do.KubernetesVersion {
	sort.Slice(versions, func(i, j int) bool { return less(versions[i], versions[j]) })
	return versions
}

func parseTaints(rawTaints []string) ([]godo.Taint, error) {
	taints := make([]godo.Taint, 0, len(rawTaints))
	for _, rawTaint := range rawTaints {
		taint, err := parseTaint(rawTaint)
		if err != nil {
			return nil, err
		}

		taints = append(taints, taint)
	}

	return taints, nil
}

func parseTaint(rawTaint string) (godo.Taint, error) {
	var key, value, effect string

	parts := strings.Split(rawTaint, ":")
	if len(parts) != 2 {
		return godo.Taint{}, fmt.Errorf("taint %q does not have a single colon separator", rawTaint)
	}

	keyValueParts := strings.Split(parts[0], "=")
	if len(keyValueParts) > 2 {
		return godo.Taint{}, fmt.Errorf("key/value part in taint %q must not consist of more than one equal sign", rawTaint)
	}
	key = keyValueParts[0]
	if len(keyValueParts) == 2 {
		value = keyValueParts[1]
	}
	effect = parts[1]

	return godo.Taint{
		Key:    key,
		Value:  value,
		Effect: effect,
	}, nil
}

func boolPtr(val bool) *bool {
	return &val
}

func intPtr(val int) *int {
	return &val
}
