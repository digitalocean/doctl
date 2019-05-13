package displayers

import (
	"io"
	"strings"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

type KubernetesClusters struct {
	KubernetesClusters do.KubernetesClusters
	Short              bool
}

var _ Displayable = &KubernetesClusters{}

func (clusters *KubernetesClusters) JSON(out io.Writer) error {
	return writeJSON(clusters.KubernetesClusters, out)
}

func (clusters *KubernetesClusters) Cols() []string {
	if clusters.Short {
		return []string{
			"ID",
			"Name",
			"Region",
			"Version",
			"AutoUpgrade",
			"Status",
			"NodePools",
		}
	}
	return []string{
		"ID",
		"Name",
		"Region",
		"Version",
		"AutoUpgrade",
		"Status",
		"Endpoint",
		"IPv4",
		"ClusterSubnet",
		"ServiceSubnet",
		"Tags",
		"Created",
		"Updated",
		"NodePools",
	}
}

func (clusters *KubernetesClusters) ColMap() map[string]string {
	if clusters.Short {
		return map[string]string{
			"ID":          "ID",
			"Name":        "Name",
			"Region":      "Region",
			"Version":     "Version",
			"AutoUpgrade": "Auto Upgrade",
			"Status":      "Status",
			"NodePools":   "Node Pools",
		}
	}
	return map[string]string{
		"ID":            "ID",
		"Name":          "Name",
		"Region":        "Region",
		"Version":       "Version",
		"AutoUpgrade":   "Auto Upgrade",
		"ClusterSubnet": "Cluster Subnet",
		"ServiceSubnet": "Service Subnet",
		"IPv4":          "IPv4",
		"Endpoint":      "Endpoint",
		"Tags":          "Tags",
		"Status":        "Status",
		"Created":       "Created At",
		"Updated":       "Updated At",
		"NodePools":     "Node Pools",
	}
}

func (clusters *KubernetesClusters) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(clusters.KubernetesClusters))

	for _, cluster := range clusters.KubernetesClusters {
		tags := strings.Join(cluster.Tags, ",")
		nodePools := make([]string, 0, len(cluster.NodePools))
		for _, pool := range cluster.NodePools {
			nodePools = append(nodePools, pool.Name)
		}
		if cluster.Status == nil {
			cluster.Status = new(godo.KubernetesClusterStatus)
		}

		o := map[string]interface{}{
			"ID":            cluster.ID,
			"Name":          cluster.Name,
			"Region":        cluster.RegionSlug,
			"Version":       cluster.VersionSlug,
			"AutoUpgrade":   cluster.AutoUpgrade,
			"ClusterSubnet": cluster.ClusterSubnet,
			"ServiceSubnet": cluster.ServiceSubnet,
			"IPv4":          cluster.IPv4,
			"Endpoint":      cluster.Endpoint,
			"Tags":          tags,
			"Status":        cluster.Status.State,
			"Created":       cluster.CreatedAt,
			"Updated":       cluster.UpdatedAt,
			"NodePools":     strings.Join(nodePools, " "),
		}
		out = append(out, o)
	}

	return out
}

type KubernetesNodePools struct {
	KubernetesNodePools do.KubernetesNodePools
}

var _ Displayable = &KubernetesNodePools{}

func (nodePools *KubernetesNodePools) JSON(out io.Writer) error {
	return writeJSON(nodePools.KubernetesNodePools, out)
}

func (nodePools *KubernetesNodePools) Cols() []string {
	return []string{
		"ID",
		"Name",
		"Size",
		"Count",
		"Tags",
		"Nodes",
	}
}

func (nodePools *KubernetesNodePools) ColMap() map[string]string {
	return map[string]string{
		"ID":    "ID",
		"Name":  "Name",
		"Size":  "Size",
		"Count": "Count",
		"Tags":  "Tags",
		"Nodes": "Nodes",
	}
}

func (nodePools *KubernetesNodePools) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(nodePools.KubernetesNodePools))

	for _, nodePools := range nodePools.KubernetesNodePools {
		tags := strings.Join(nodePools.Tags, ",")
		nodes := make([]string, 0, len(nodePools.Nodes))
		for _, node := range nodePools.Nodes {
			nodes = append(nodes, node.Name)
		}

		o := map[string]interface{}{
			"ID":    nodePools.ID,
			"Name":  nodePools.Name,
			"Size":  nodePools.Size,
			"Count": nodePools.Count,
			"Tags":  tags,
			"Nodes": nodes,
		}
		out = append(out, o)
	}

	return out
}

type KubernetesVersions struct {
	KubernetesVersions do.KubernetesVersions
}

var _ Displayable = &KubernetesVersions{}

func (versions *KubernetesVersions) JSON(out io.Writer) error {
	return writeJSON(versions.KubernetesVersions, out)
}

func (versions *KubernetesVersions) Cols() []string {
	return []string{
		"Slug",
		"KubernetesVersion",
	}
}

func (versions *KubernetesVersions) ColMap() map[string]string {
	return map[string]string{
		"Slug":              "Slug",
		"KubernetesVersion": "Kubernetes Version",
	}
}

func (versions *KubernetesVersions) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(versions.KubernetesVersions))

	for _, version := range versions.KubernetesVersions {

		o := map[string]interface{}{
			"Slug":              version.KubernetesVersion.Slug,
			"KubernetesVersion": version.KubernetesVersion.KubernetesVersion,
		}
		out = append(out, o)
	}

	return out
}

type KubernetesRegions struct {
	KubernetesRegions do.KubernetesRegions
}

var _ Displayable = &KubernetesRegions{}

func (regions *KubernetesRegions) JSON(out io.Writer) error {
	return writeJSON(regions.KubernetesRegions, out)
}

func (regions *KubernetesRegions) Cols() []string {
	return []string{
		"Slug",
		"Name",
	}
}

func (regions *KubernetesRegions) ColMap() map[string]string {
	return map[string]string{
		"Slug": "Slug",
		"Name": "Name",
	}
}

func (regions *KubernetesRegions) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(regions.KubernetesRegions))

	for _, region := range regions.KubernetesRegions {

		o := map[string]interface{}{
			"Slug": region.KubernetesRegion.Slug,
			"Name": region.KubernetesRegion.Name,
		}
		out = append(out, o)
	}

	return out
}

type KubernetesNodeSizes struct {
	KubernetesNodeSizes do.KubernetesNodeSizes
}

var _ Displayable = &KubernetesNodeSizes{}

func (nodeSizes *KubernetesNodeSizes) JSON(out io.Writer) error {
	return writeJSON(nodeSizes.KubernetesNodeSizes, out)
}

func (nodeSizes *KubernetesNodeSizes) Cols() []string {
	return []string{
		"Slug",
		"Name",
	}
}

func (nodeSizes *KubernetesNodeSizes) ColMap() map[string]string {
	return map[string]string{
		"Slug": "Slug",
		"Name": "Name",
	}
}

func (nodeSizes *KubernetesNodeSizes) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(nodeSizes.KubernetesNodeSizes))

	for _, size := range nodeSizes.KubernetesNodeSizes {

		o := map[string]interface{}{
			"Slug": size.KubernetesNodeSize.Slug,
			"Name": size.KubernetesNodeSize.Name,
		}
		out = append(out, o)
	}

	return out
}
