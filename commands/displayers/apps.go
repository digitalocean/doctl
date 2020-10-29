package displayers

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
)

type Apps []*godo.App

var _ Displayable = (*Apps)(nil)

func (a Apps) Cols() []string {
	return []string{
		"ID",
		"Spec.Name",
		"DefaultIngress",
		"ActiveDeployment.ID",
		"InProgressDeployment.ID",
		"Created",
		"Updated",
	}
}

func (a Apps) ColMap() map[string]string {
	return map[string]string{
		"ID":                      "ID",
		"Spec.Name":               "Spec Name",
		"DefaultIngress":          "Default Ingress",
		"ActiveDeployment.ID":     "Active Deployment ID",
		"InProgressDeployment.ID": "In Progress Deployment ID",
		"Created":                 "Created At",
		"Updated":                 "Updated At",
	}
}

func (a Apps) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, len(a))

	for i, app := range a {
		var (
			activeDeploymentID     string
			inProgressDeploymentID string
		)

		if app.ActiveDeployment != nil {
			activeDeploymentID = app.ActiveDeployment.ID
		}

		if app.InProgressDeployment != nil {
			inProgressDeploymentID = app.InProgressDeployment.ID
		}

		out[i] = map[string]interface{}{
			"ID":                      app.ID,
			"Spec.Name":               app.Spec.Name,
			"DefaultIngress":          app.DefaultIngress,
			"ActiveDeployment.ID":     activeDeploymentID,
			"InProgressDeployment.ID": inProgressDeploymentID,
			"Created":                 app.CreatedAt,
			"Updated":                 app.UpdatedAt,
		}
	}
	return out
}

func (a Apps) JSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(a)
}

type Deployments []*godo.Deployment

var _ Displayable = (*Deployments)(nil)

func (d Deployments) Cols() []string {
	return []string{
		"ID",
		"Cause",
		"Progress",
		"Created",
		"Updated",
	}
}

func (d Deployments) ColMap() map[string]string {
	return map[string]string{
		"ID":       "ID",
		"Cause":    "Cause",
		"Progress": "Progress",
		"Created":  "Created At",
		"Updated":  "Updated At",
	}
}

func (d Deployments) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, len(d))

	for i, deployment := range d {
		progress := fmt.Sprintf("%d/%d", deployment.Progress.SuccessSteps, deployment.Progress.TotalSteps)
		if deployment.Progress.ErrorSteps > 0 {
			progress = fmt.Sprintf("%s (errors: %d)", progress, deployment.Progress.ErrorSteps)
		}

		out[i] = map[string]interface{}{
			"ID":       deployment.ID,
			"Cause":    deployment.Cause,
			"Progress": progress,
			"Created":  deployment.CreatedAt,
			"Updated":  deployment.UpdatedAt,
		}
	}
	return out
}

func (d Deployments) JSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(d)
}

type AppRegions []*godo.AppRegion

var _ Displayable = (*AppRegions)(nil)

func (r AppRegions) Cols() []string {
	return []string{
		"Slug",
		"Label",
		"Continent",
		"DataCenters",
		"Disabled",
		"Reason",
		"Default",
	}
}

func (r AppRegions) ColMap() map[string]string {
	return map[string]string{
		"Slug":        "Region",
		"Label":       "Label",
		"Continent":   "Continent",
		"DataCenters": "Data Centers",
		"Disabled":    "Is Disabled?",
		"Reason":      "Reason (if disabled)",
		"Default":     "Is Default?",
	}
}

func (r AppRegions) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, len(r))

	for i, region := range r {
		out[i] = map[string]interface{}{
			"Slug":        region.Slug,
			"Label":       region.Label,
			"Continent":   region.Continent,
			"DataCenters": region.DataCenters,
			"Disabled":    region.Disabled,
			"Reason":      region.Reason,
			"Default":     region.Default,
		}
	}
	return out
}

func (r AppRegions) JSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(r)
}

type AppTiers []*godo.AppTier

var _ Displayable = (*AppTiers)(nil)

func (t AppTiers) Cols() []string {
	return []string{
		"Name",
		"Slug",
		"EgressBandwidthBytes",
		"BuildSeconds",
	}
}

func (t AppTiers) ColMap() map[string]string {
	return map[string]string{
		"Name":                 "Name",
		"Slug":                 "Slug",
		"EgressBandwidthBytes": "Egress Bandwidth",
		"BuildSeconds":         "Build Seconds",
	}
}

func (t AppTiers) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, len(t))

	for i, tier := range t {
		egressBandwidth, _ := strconv.ParseUint(tier.EgressBandwidthBytes, 10, 64)
		out[i] = map[string]interface{}{
			"Name":                 tier.Name,
			"Slug":                 tier.Slug,
			"EgressBandwidthBytes": BytesToHumanReadibleUnit(egressBandwidth),
			"BuildSeconds":         tier.BuildSeconds,
		}
	}
	return out
}

func (t AppTiers) JSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(t)
}

type AppInstanceSizes []*godo.AppInstanceSize

var _ Displayable = (*AppInstanceSizes)(nil)

func (is AppInstanceSizes) Cols() []string {
	return []string{
		"Name",
		"Slug",
		"CPUs",
		"Memory",
		"USDPerMonth",
		"USDPerSecond",
		"TierSlug",
		"TierUpgradeDowngradePath",
	}
}

func (is AppInstanceSizes) ColMap() map[string]string {
	return map[string]string{
		"Name":                     "Name",
		"Slug":                     "Slug",
		"CPUs":                     "CPUs",
		"Memory":                   "Memory",
		"USDPerMonth":              "$/month",
		"USDPerSecond":             "$/second",
		"TierSlug":                 "Tier",
		"TierUpgradeDowngradePath": "Tier Downgrade/Upgrade Path",
	}
}

func (is AppInstanceSizes) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, len(is))

	for i, instanceSize := range is {
		memory, _ := strconv.ParseUint(instanceSize.MemoryBytes, 10, 64)
		cpus := fmt.Sprintf("%s %s", instanceSize.CPUs, strings.ToLower(string(instanceSize.CPUType)))
		usdPerSecond, _ := strconv.ParseFloat(instanceSize.USDPerSecond, 64)

		var upgradeDowngradePath string
		if instanceSize.TierDowngradeTo != "" {
			upgradeDowngradePath = instanceSize.TierDowngradeTo + " <- "
		}
		upgradeDowngradePath = upgradeDowngradePath + instanceSize.Slug
		if instanceSize.TierUpgradeTo != "" {
			upgradeDowngradePath = upgradeDowngradePath + " -> " + instanceSize.TierUpgradeTo
		}

		out[i] = map[string]interface{}{
			"Name":                     instanceSize.Name,
			"Slug":                     instanceSize.Slug,
			"CPUs":                     cpus,
			"Memory":                   BytesToHumanReadibleUnit(memory),
			"USDPerMonth":              instanceSize.USDPerMonth,
			"USDPerSecond":             fmt.Sprintf("%.7f", usdPerSecond),
			"TierSlug":                 instanceSize.TierSlug,
			"TierUpgradeDowngradePath": upgradeDowngradePath,
		}
	}
	return out
}

func (is AppInstanceSizes) JSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(is)
}
