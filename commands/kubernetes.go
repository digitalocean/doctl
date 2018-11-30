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
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/pborman/uuid"
	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const maxAPIFailures = 3

func errNoClusterByName(name string) error {
	return fmt.Errorf("no cluster goes by the name %q", name)
}

func errAmbigousClusterName(name string, ids []string) error {
	return fmt.Errorf("many clusters go by the name %q, they have the following IDs: %v", name, ids)
}

func errNoPoolByName(name string) error {
	return fmt.Errorf("no node pool goes by the name %q", name)
}

func errAmbigousPoolName(name string, ids []string) error {
	return fmt.Errorf("many node pools go by the name %q, they have the following IDs: %v", name, ids)
}

func errNoClusterNodeByName(name string) error {
	return fmt.Errorf("no node goes by the name %q", name)
}

func errAmbigousClusterNodeName(name string, ids []string) error {
	return fmt.Errorf("many nodes go by the name %q, they have the following IDs: %v", name, ids)
}

// Kubernetes creates the kubernetes command.
func Kubernetes() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "kubernetes",
			Aliases: []string{"kube", "k8s", "k"},
			Short:   "[beta] kubernetes commands",
			Long:    "[beta] kubernetes is used to access Kubernetes commands",
			Hidden:  !isBeta(),
		},
	}

	cmd.AddCommand(kubernetesCluster())
	cmd.AddCommand(kubernetesOptions())
	return cmd
}

func kubernetesCluster() *Command {

	const (
		defaultNodeSize  = "s-1vcpu-1gb"
		defaultNodeCount = 3
		defaultRegion    = "nyc1"
	)

	cmd := &Command{
		Command: &cobra.Command{
			Use:     "cluster",
			Aliases: []string{"clusters", "c"},
			Short:   "clusters commands",
			Long:    "clusters is used to access commands on Kubernetes clusters",
		},
	}

	cmd.AddCommand(kubernetesKubeconfig())

	cmd.AddCommand(kubernetesNodePools())

	CmdBuilder(cmd, RunKubernetesClusterGet, "get <id|name>", "get a cluster", Writer, aliasOpt("g"))
	CmdBuilder(cmd, RunKubernetesClusterList, "list", "get a list of your clusters", Writer, aliasOpt("ls"))

	cmdKubeClusterCreate := CmdBuilder(cmd, RunKubernetesClusterCreate(defaultNodeSize, defaultNodeCount), "create <name>", "create a cluster", Writer, aliasOpt("c"))
	AddStringFlag(cmdKubeClusterCreate, doctl.ArgRegionSlug, "", defaultRegion, "cluster region location, example value: nyc1", requiredOpt())
	AddStringFlag(cmdKubeClusterCreate, doctl.ArgClusterVersionSlug, "", "", "cluster version")
	AddStringSliceFlag(cmdKubeClusterCreate, doctl.ArgTagNames, "", nil, "cluster tags")
	AddStringFlag(cmdKubeClusterCreate, doctl.ArgSizeSlug, "", defaultNodeSize, "size of the nodes in the default node pool (incompatible with --"+doctl.ArgClusterNodePool+")")
	AddStringFlag(cmdKubeClusterCreate, doctl.ArgNodePoolCount, "", strconv.Itoa(defaultNodeCount), "size of the nodes in the default node pool (incompatible with --"+doctl.ArgClusterNodePool+")")
	AddStringSliceFlag(cmdKubeClusterCreate, doctl.ArgClusterNodePool, "", nil, `cluster node pools in the form "name=your-name;size=droplet_size;count=5;tag=tag1;tag=tag2"`, requiredOpt())
	AddBoolFlag(cmdKubeClusterCreate, doctl.ArgClusterUpdateKubeconfig, "", true, "whether to add the created cluster to your kubeconfig")
	AddBoolFlag(cmdKubeClusterCreate, doctl.ArgCommandWait, "", true, "whether to wait for the created cluster become running")

	cmdKubeClusterUpdate := CmdBuilder(cmd, RunKubernetesClusterUpdate, "update <id|name>", "update a cluster's properties", Writer, aliasOpt("u"))
	AddStringFlag(cmdKubeClusterUpdate, doctl.ArgClusterName, "", "", "new cluster name")
	AddStringSliceFlag(cmdKubeClusterUpdate, doctl.ArgTagNames, "", nil, "new cluster tags")
	AddBoolFlag(cmdKubeClusterUpdate, doctl.ArgClusterUpdateKubeconfig, "", true, "whether to update the cluster in your kubeconfig")

	cmdKubeClusterDelete := CmdBuilder(cmd, RunKubernetesClusterDelete, "delete <id|name>", "delete a cluster", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdKubeClusterDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force cluster delete")
	AddBoolFlag(cmdKubeClusterDelete, doctl.ArgClusterUpdateKubeconfig, "", true, "whether to remove the deleted cluster to your kubeconfig")

	return cmd
}

func kubernetesKubeconfig() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "kubeconfig",
			Aliases: []string{"kubecfg", "k8scfg", "config", "cfg"},
			Short:   "kubeconfig commands",
			Long:    "kubeconfig commands are used retrieve a cluster's credentials and manipulate them",
		},
	}

	CmdBuilder(cmd, RunKubernetesKubeconfigPrint, "print <cluster-id|cluster-name>", "print a cluster's kubeconfig to standard out", Writer, aliasOpt("p", "g"))
	CmdBuilder(cmd, RunKubernetesKubeconfigSave, "save <cluster-id|cluster-name>", "save a cluster's credentials to your local kubeconfig", Writer, aliasOpt("s"))
	CmdBuilder(cmd, RunKubernetesKubeconfigRemove, "remove <cluster-id|cluster-name>", "remove a cluster's credentials from your local kubeconfig", Writer, aliasOpt("d", "rm"))
	return cmd
}

func kubernetesNodePools() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "node-pool",
			Aliases: []string{"node-pools", "nodepool", "nodepools", "pool", "pools", "np", "p"},
			Short:   "node pool commands",
			Long:    "node pool commands are used to act on a cluster's node pools",
		},
	}

	CmdBuilder(cmd, RunKubernetesNodePoolGet, "get <cluster-id|cluster-name> <pool-id|pool-name>", "get a cluster's node pool", Writer, aliasOpt("g"))
	CmdBuilder(cmd, RunKubernetesNodePoolList, "list <cluster-id|cluster-name>", "list a cluster's node pools", Writer, aliasOpt("ls"))

	cmdKubeNodePoolCreate := CmdBuilder(cmd, RunKubernetesNodePoolCreate, "create <cluster-id|cluster-name>", "create a new node pool for a cluster", Writer, aliasOpt("c"))
	AddStringFlag(cmdKubeNodePoolCreate, doctl.ArgNodePoolName, "", "", "node pool name", requiredOpt())
	AddStringFlag(cmdKubeNodePoolCreate, doctl.ArgSizeSlug, "", "", "size of nodes in the node pool", requiredOpt())
	AddStringFlag(cmdKubeNodePoolCreate, doctl.ArgNodePoolCount, "", "", "count of nodes in the node pool", requiredOpt())
	AddStringFlag(cmdKubeNodePoolCreate, doctl.ArgTagNames, "", "", "tags to apply to the node pool")

	cmdKubeNodePoolUpdate := CmdBuilder(cmd, RunKubernetesNodePoolUpdate, "update <cluster-id|cluster-name> <pool-id|pool-name>", "update an existing node pool in a cluster", Writer, aliasOpt("u"))
	AddStringFlag(cmdKubeNodePoolUpdate, doctl.ArgNodePoolName, "", "", "node pool name")
	AddStringFlag(cmdKubeNodePoolUpdate, doctl.ArgNodePoolCount, "", "", "count of nodes in the node pool")
	AddStringFlag(cmdKubeNodePoolUpdate, doctl.ArgTagNames, "", "", "tags to apply to the node pool")

	cmdKubeNodePoolRecycle := CmdBuilder(cmd, RunKubernetesNodePoolRecycle, "recycle <cluster-id|cluster-name> <pool-id|pool-name>", "recycle nodes in a node pool", Writer, aliasOpt("r"))
	AddStringFlag(cmdKubeNodePoolRecycle, doctl.ArgNodePoolNodeIDs, "", "", "ID or name of the nodes in the node pool to recycle")

	cmdKubeNodePoolDelete := CmdBuilder(cmd, RunKubernetesNodePoolDelete, "delete <cluster-id|cluster-name> <pool-id|pool-name>", "delete node pool from a cluster", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdKubeNodePoolDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force node pool delete")
	return cmd
}

func kubernetesOptions() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "options",
			Aliases: []string{"opts", "o"},
			Short:   "options commands",
			Long:    "options commands are used to find options for Kubernetes clusters",
		},
	}

	CmdBuilder(cmd, RunKubeOptionsListVersion, "versions", "versions that can be used to create a Kubernetes cluster", Writer, aliasOpt("v"))
	return cmd
}

// Clusters

// RunKubernetesClusterGet retrieves an existing kubernetes by its identifier.
func RunKubernetesClusterGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterIDorName := c.Args[0]

	cluster, err := clusterByIDorName(c.Kubernetes(), clusterIDorName)
	if err != nil {
		return err
	}
	return displayClusters(c, false, *cluster)
}

// RunKubernetesClusterList lists kubernetess.
func RunKubernetesClusterList(c *CmdConfig) error {
	kube := c.Kubernetes()
	list, err := kube.List()
	if err != nil {
		return err
	}

	return displayClusters(c, true, list...)
}

// RunKubernetesClusterCreate creates a new kubernetes with a given configuration.
func RunKubernetesClusterCreate(defaultNodeSize string, defaultNodeCount int) func(*CmdConfig) error {
	return func(c *CmdConfig) error {
		if len(c.Args) != 1 {
			return doctl.NewMissingArgsErr(c.NS)
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

		kube := c.Kubernetes()

		cluster, err := kube.Create(r)
		if err != nil {
			return err
		}

		if update {
			notice("cluster created, fetching credentials")
			tryUpdateKubeconfig(kube, cluster.ID)
		}

		if wait {
			notice("cluster is provisioning, waiting for cluster to be running")
			cluster, err = waitForClusterRunning(kube, cluster.ID)
			if err != nil {
				warn("cluster didn't become running: %v", err)
			}
		}

		return displayClusters(c, true, *cluster)
	}
}

// RunKubernetesClusterUpdate updates an existing kubernetes with new configuration.
func RunKubernetesClusterUpdate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	update, err := c.Doit.GetBool(c.NS, doctl.ArgClusterUpdateKubeconfig)
	if err != nil {
		return err
	}
	clusterID, err := clusterIDize(c.Kubernetes(), c.Args[0])
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
		notice("cluster updated, fetching new credentials")
		tryUpdateKubeconfig(kube, clusterID)
	}

	return displayClusters(c, true, *cluster)
}

func tryUpdateKubeconfig(kube do.KubernetesService, clusterID string) {
	var (
		kubeconfig []byte
		err        error
	)
	for tries := 0; ; tries++ {
		kubeconfig, err = kube.GetKubeConfig(clusterID)
		if err == nil {
			break
		}
		if tries >= maxAPIFailures {
			warn("couldn't get credentials for cluster, it will not be added to your kubeconfig: %v", err)
			return
		}
		time.Sleep(2 * time.Second)
	}
	if err := writeOrAddToKubeconfig(kubeconfig); err != nil {
		warn("couldn't write cluster credentials: %v", err)
	}
}

// RunKubernetesClusterDelete deletes a kubernetes by its identifier.
func RunKubernetesClusterDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	update, err := c.Doit.GetBool(c.NS, doctl.ArgClusterUpdateKubeconfig)
	if err != nil {
		return err
	}
	clusterID, err := clusterIDize(c.Kubernetes(), c.Args[0])
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirm("delete this Kubernetes cluster") == nil {
		// continue
	} else {
		return fmt.Errorf("operation aborted")
	}
	kube := c.Kubernetes()

	var kubeconfig []byte
	if update {
		// get the cluster's kubeconfig before issuing the delete, so that we can
		// cleanup the entry from the local file
		kubeconfig, err = kube.GetKubeConfig(clusterID)
		if err != nil {
			warn("couldn't get credentials for cluster, it will not be remove from your kubeconfig")
		}
	}
	if err := kube.Delete(clusterID); err != nil {
		return err
	}
	if kubeconfig != nil {
		notice("cluster deleted, removing credentials")
		if err := removeFromKubeconfig(kubeconfig); err != nil {
			warn("Cluster was deleted, but we couldn't remove it from your local kubeconfig. Try doing it manually.")
		}
	}

	return nil
}

// Kubeconfig

// RunKubernetesKubeconfigPrint retrieves an existing kubernetes config and prints it.
func RunKubernetesKubeconfigPrint(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	kube := c.Kubernetes()
	clusterID, err := clusterIDize(kube, c.Args[0])
	if err != nil {
		return err
	}
	kubeconfig, err := kube.GetKubeConfig(clusterID)
	if err != nil {
		return err
	}
	_, err = c.Out.Write(kubeconfig)
	return err
}

// RunKubernetesKubeconfigSave retrieves an existing kubernetes config and saves it to your local kubeconfig.
func RunKubernetesKubeconfigSave(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	kube := c.Kubernetes()
	clusterID, err := clusterIDize(kube, c.Args[0])
	if err != nil {
		return err
	}
	kubeconfig, err := kube.GetKubeConfig(clusterID)
	if err != nil {
		return err
	}
	if err := writeOrAddToKubeconfig(kubeconfig); err != nil {
		return err
	}
	return nil
}

// RunKubernetesKubeconfigRemove retrieves an existing kubernetes config and removes it from your local kubeconfig.
func RunKubernetesKubeconfigRemove(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	kube := c.Kubernetes()
	clusterID, err := clusterIDize(kube, c.Args[0])
	if err != nil {
		return err
	}
	kubeconfig, err := kube.GetKubeConfig(clusterID)
	if err != nil {
		return err
	}
	if err := removeFromKubeconfig(kubeconfig); err != nil {
		return err
	}
	return nil
}

// Node Pools

// RunKubernetesNodePoolGet retrieves an existing cluster node pool by its identifier.
func RunKubernetesNodePoolGet(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c.Kubernetes(), c.Args[0])
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
func RunKubernetesNodePoolList(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c.Kubernetes(), c.Args[0])
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
func RunKubernetesNodePoolCreate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c.Kubernetes(), c.Args[0])
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
func RunKubernetesNodePoolUpdate(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c.Kubernetes(), c.Args[0])
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

// RunKubernetesNodePoolRecycle recycles an existing kubernetes with new configuration.
func RunKubernetesNodePoolRecycle(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c.Kubernetes(), c.Args[0])
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

// RunKubernetesNodePoolDelete deletes a kubernetes by its identifier.
func RunKubernetesNodePoolDelete(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	clusterID, err := clusterIDize(c.Kubernetes(), c.Args[0])
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
	if force || AskForConfirm("delete this Kubernetes node pool") == nil {
		kube := c.Kubernetes()
		if err := kube.DeleteNodePool(clusterID, poolID); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("operation aborted")
	}
	return nil
}

// RunKubeOptionsListVersion deletes a kubernetes by its identifier.
func RunKubeOptionsListVersion(c *CmdConfig) error {
	kube := c.Kubernetes()
	versions, err := kube.GetVersions()
	if err != nil {
		return err
	}
	item := &displayers.KubernetesVersions{KubernetesVersions: versions}
	return c.Display(item)
}

func buildClusterCreateRequestFromArgs(c *CmdConfig, r *godo.KubernetesClusterCreateRequest, defaultNodeSize string, defaultNodeCount int) error {
	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	r.RegionSlug = region

	version, err := getVersionOrLatest(c)
	if err != nil {
		return err
	}
	r.VersionSlug = version

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagNames)
	if err != nil {
		return err
	}
	r.Tags = tags

	// node pools

	nodePoolSize, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return err
	}

	nodePoolCount, err := c.Doit.GetInt(c.NS, doctl.ArgNodePoolCount)
	if err != nil {
		return err
	}

	nodePoolSpecs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgClusterNodePool)
	if err != nil {
		return err
	}

	switch {
	case len(nodePoolSpecs) != 0 && (nodePoolSize != "" || nodePoolCount != 0):
		return fmt.Errorf("flags %q and %q cannot be provided when %q is present", doctl.ArgSizeSlug, doctl.ArgNodePoolCount, doctl.ArgClusterNodePool)
	case len(nodePoolSpecs) != 0:
		nodePools, err := buildNodePoolCreateRequestsFromArgs(c, nodePoolSpecs, r.Name, defaultNodeSize, defaultNodeCount)
		if err != nil {
			return err
		}
		r.NodePools = nodePools
	default:
		nodePoolName := r.Name + "-default-pool"
		if nodePoolSize == "" {
			nodePoolSize = defaultNodeSize
		}
		if nodePoolCount == 0 {
			nodePoolCount = defaultNodeCount
		}
		r.NodePools = []*godo.KubernetesNodePoolCreateRequest{{
			Name:  nodePoolName,
			Size:  nodePoolSize,
			Count: nodePoolCount,
		}}
	}

	return nil
}

func buildClusterUpdateRequestFromArgs(c *CmdConfig, r *godo.KubernetesClusterUpdateRequest) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgClusterName)
	if err != nil {
		return err
	}
	r.Name = name

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagNames)
	if err != nil {
		return err
	}
	r.Tags = tags

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
		// at least some of the args weren't UUIDs, so assume that they're all names
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
			return nil, fmt.Errorf("invalid node pool arguments for flag %d: %v", i, err)
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
		Name:  defaultName,
		Size:  defaultSize,
		Count: defaultCount,
	}
	for _, arg := range strings.Split(nodePool, argSeparator) {
		kvs := strings.SplitN(arg, kvSeparator, 2)
		if len(kvs) < 2 {
			return nil, fmt.Errorf("a node pool string argument must be of the form `key=value`, got KVs %v", kvs)
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
				return nil, errors.New("node pool count argument must be a valid integer")
			}
			out.Count = int(count)
		case "tag":
			out.Tags = append(out.Tags, value)
		default:
			return nil, fmt.Errorf("unsupported node pool argument %q", key)
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

	count, err := c.Doit.GetInt(c.NS, doctl.ArgNodePoolCount)
	if err != nil {
		return err
	}
	r.Count = count

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagNames)
	if err != nil {
		return err
	}
	r.Tags = tags

	return nil
}

func buildNodePoolUpdateRequestFromArgs(c *CmdConfig, r *godo.KubernetesNodePoolUpdateRequest) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgNodePoolName)
	if err != nil {
		return err
	}
	r.Name = name

	count, err := c.Doit.GetInt(c.NS, doctl.ArgNodePoolCount)
	if err != nil {
		return err
	}
	r.Count = count

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagNames)
	if err != nil {
		return err
	}
	r.Tags = tags

	return nil
}

func writeOrAddToKubeconfig(kubeconfig []byte) error {
	remote, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return err
	}
	kubectlDefaults := clientcmd.NewDefaultPathOptions()
	currentConfig, err := kubectlDefaults.GetStartingConfig()
	if err != nil {
		return err
	}
	notice("adding cluster credentials to kubeconfig file found in %q", kubectlDefaults.GlobalFile)
	if err := mergeKubeconfig(remote, currentConfig); err != nil {
		return fmt.Errorf("couldn't use the kubeconfig info received, %v", err)
	}
	currentConfig.CurrentContext = remote.CurrentContext
	notice("current kubectl context changed to %q", currentConfig.CurrentContext)
	return clientcmd.ModifyConfig(kubectlDefaults, *currentConfig, false)
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
	notice("removing cluster credentials from kubeconfig file found in %q", kubectlDefaults.GlobalFile)
	if err := removeKubeconfig(remote, currentConfig); err != nil {
		return fmt.Errorf("couldn't use the kubeconfig info received, %v", err)
	}
	return clientcmd.ModifyConfig(kubectlDefaults, *currentConfig, false)
}

// mergeKubeconfig merges a remote cluster's config file with a local config file,
// assuming that the current context in the remote config file points to the
// cluster details to add to the local config.
func mergeKubeconfig(remote, local *clientcmdapi.Config) error {
	remoteCtx, ok := remote.Contexts[remote.CurrentContext]
	if !ok {
		// this is a bug in the backend, we received incomplete/non-sensical data
		return fmt.Errorf("the remote config has no context entry named %q. This is a bug, please open a ticket with DigitalOcean!",
			remote.CurrentContext,
		)
	}
	remoteCluster, ok := remote.Clusters[remoteCtx.Cluster]
	if !ok {
		// this is a bug in the backend, we received incomplete/non-sensical data
		return fmt.Errorf("the remote config has no cluster entry named %q. This is a bug, please open a ticket with DigitalOcean!",
			remoteCtx.Cluster,
		)
	}
	remoteAuthInfo, ok := remote.AuthInfos[remoteCtx.AuthInfo]
	if !ok {
		// this is a bug in the backend, we received incomplete/non-sensical data
		return fmt.Errorf("the remote config has no user entry named %q. This is a bug, please open a ticket with DigitalOcean!",
			remoteCtx.AuthInfo,
		)
	}

	local.Contexts[remote.CurrentContext] = remoteCtx
	local.Clusters[remoteCtx.Cluster] = remoteCluster
	local.AuthInfos[remoteCtx.AuthInfo] = remoteAuthInfo
	return nil
}

// removeKubeconfig removes a remote cluster's config file from a local config file,
// assuming that the current context in the remote config file points to the
// cluster details to reomve from the local config.
func removeKubeconfig(remote, local *clientcmdapi.Config) error {
	remoteCtx, ok := remote.Contexts[remote.CurrentContext]
	if !ok {
		// this is a bug in the backend, we received incomplete/non-sensical data
		return fmt.Errorf("the remote config has no context entry named %q. This is a bug, please open a ticket with DigitalOcean!",
			remote.CurrentContext,
		)
	}

	delete(local.Contexts, remote.CurrentContext)
	delete(local.Clusters, remoteCtx.Cluster)
	delete(local.AuthInfos, remoteCtx.AuthInfo)
	if local.CurrentContext == remote.CurrentContext {
		local.CurrentContext = ""
		notice("cluster was set as current context for kubectl. It has been removed, you might want to set a new one.")
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
		if err != nil {
			if failCount >= maxAPIFailures {
				return nil, err
			}
			// tolerate transient API failures
			time.Sleep(time.Second)
		} else {
			failCount = 0 // API responded, reset it's error counter
		}
		if cluster.Status == nil {
			time.Sleep(1 * time.Second)
			continue
		}
		switch cluster.Status.State {
		case godo.KubernetesClusterStatusRunning:
			return cluster, nil
		case godo.KubernetesClusterStatusProvisioning:
			time.Sleep(5 * time.Second)
		default:
			return cluster, fmt.Errorf("unknown status: [%s]", cluster.Status.State)
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
		return nil, errAmbigousClusterName(idOrName, ids)
	default:
		if len(out) != 1 {
			panic("the default case should always have len(out) == 1")
		}
		return out[0], nil
	}
}

// clusterIDize attempts to make a cluster ID/name string be a cluster ID.
// use this as opposed to `clusterByIDorName` if you just care about getting
// a cluster ID and don't need the cluster object itself
func clusterIDize(kube do.KubernetesService, idOrName string) (string, error) {
	if looksLikeUUID(idOrName) {
		return idOrName, nil
	}
	clusters, err := kube.List()
	if err != nil {
		return "", err
	}
	var ids []string
	for _, c := range clusters {
		if c.Name == idOrName {
			id := c.ID
			ids = append(ids, id)
		}
	}
	switch {
	case len(ids) == 0:
		return "", errNoClusterByName(idOrName)
	case len(ids) > 1:
		return "", errAmbigousClusterName(idOrName, ids)
	default:
		if len(ids) != 1 {
			panic("the default case should always have len(ids) == 1")
		}
		return ids[0], nil
	}
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
		return nil, errAmbigousPoolName(idOrName, ids)
	default:
		if len(out) != 1 {
			panic("the default case should always have len(out) == 1")
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
		return "", errAmbigousPoolName(idOrName, ids)
	default:
		if len(ids) != 1 {
			panic("the default case should always have len(ids) == 1")
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
	var out []*godo.KubernetesNode
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
		return nil, errAmbigousClusterNodeName(name, ids)
	default:
		if len(out) != 1 {
			panic("the default case should always have len(out) == 1")
		}
		return out[0], nil
	}
}

func looksLikeUUID(str string) bool {
	return uuid.Parse(str) != nil
}

func getVersionOrLatest(c *CmdConfig) (string, error) {
	version, err := c.Doit.GetString(c.NS, doctl.ArgClusterVersionSlug)
	if err != nil {
		return "", err
	}
	if version != "" {
		return version, nil
	}
	versions, err := c.Kubernetes().GetVersions()
	if err != nil {
		return "", fmt.Errorf("no version flag provided and unable to lookup the latest version from the API: %v", err)
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

func latestReleases(versions []do.KubernetesVersion) ([]do.KubernetesVersion, error) {
	versionsByK8S := versionMapBy(versions, func(v do.KubernetesVersion) string {
		return v.KubernetesVersion.KubernetesVersion
	})

	var out []do.KubernetesVersion
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
	for i, v := range versions[1:] {
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
