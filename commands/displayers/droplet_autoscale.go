package displayers

import (
	"io"

	"github.com/digitalocean/godo"
)

type DropletAutoscalePools struct {
	AutoscalePools []*godo.DropletAutoscalePool `json:"autoscale_pools"`
}

var _ Displayable = &DropletAutoscalePools{}

func (d *DropletAutoscalePools) Cols() []string {
	return []string{
		"ID",
		"Name",
		"Region",
		"Status",
		"Min Instance",
		"Max Instance",
		"Target Instance",
		"Avg CPU Util",
		"Avg Mem Util",
		"Target CPU Util",
		"Target Mem Util",
	}
}

func (d *DropletAutoscalePools) ColMap() map[string]string {
	return map[string]string{
		"ID":              "ID",
		"Name":            "Name",
		"Region":          "Region",
		"Status":          "Status",
		"Min Instance":    "Min Instance",
		"Max Instance":    "Max Instance",
		"Target Instance": "Target Instance",
		"Avg CPU Util":    "Avg CPU Util",
		"Avg Mem Util":    "Avg Mem Util",
		"Target CPU Util": "Target CPU Util",
		"Target Mem Util": "Target Mem Util",
	}
}

func (d *DropletAutoscalePools) KV() []map[string]any {
	out := make([]map[string]any, 0, len(d.AutoscalePools))
	for _, pool := range d.AutoscalePools {
		var cpuUtil, memUtil any
		if pool.CurrentUtilization != nil {
			cpuUtil = pool.CurrentUtilization.CPU
			memUtil = pool.CurrentUtilization.Memory
		}
		out = append(out, map[string]any{
			"ID":              pool.ID,
			"Name":            pool.Name,
			"Region":          pool.DropletTemplate.Region,
			"Status":          pool.Status,
			"Min Instance":    pool.Config.MinInstances,
			"Max Instance":    pool.Config.MaxInstances,
			"Target Instance": pool.Config.TargetNumberInstances,
			"Avg CPU Util":    cpuUtil,
			"Avg Mem Util":    memUtil,
			"Target CPU Util": pool.Config.TargetCPUUtilization,
			"Target Mem Util": pool.Config.TargetMemoryUtilization,
		})
	}
	return out
}

func (d *DropletAutoscalePools) JSON(out io.Writer) error {
	return writeJSON(d.AutoscalePools, out)
}

type DropletAutoscaleResources struct {
	Droplets []*godo.DropletAutoscaleResource `json:"droplets"`
}

var _ Displayable = &DropletAutoscaleResources{}

func (d *DropletAutoscaleResources) Cols() []string {
	return []string{
		"ID",
		"Status",
		"Health Status",
		"Unhealthy Reason",
		"CPU Util",
		"Mem Util",
	}
}

func (d *DropletAutoscaleResources) ColMap() map[string]string {
	return map[string]string{
		"ID":               "ID",
		"Status":           "Status",
		"Health Status":    "Health Status",
		"Unhealthy Reason": "Unhealthy Reason",
		"CPU Util":         "CPU Util",
		"Mem Util":         "Mem Util",
	}
}

func (d *DropletAutoscaleResources) KV() []map[string]any {
	out := make([]map[string]any, 0, len(d.Droplets))
	for _, droplet := range d.Droplets {
		var cpuUtil, memUtil any
		if droplet.CurrentUtilization != nil {
			cpuUtil = droplet.CurrentUtilization.CPU
			memUtil = droplet.CurrentUtilization.Memory
		}
		out = append(out, map[string]any{
			"ID":               droplet.DropletID,
			"Status":           droplet.Status,
			"Health Status":    droplet.HealthStatus,
			"Unhealthy Reason": droplet.UnhealthyReason,
			"CPU Util":         cpuUtil,
			"Mem Util":         memUtil,
		})
	}
	return out
}

func (d *DropletAutoscaleResources) JSON(out io.Writer) error {
	return writeJSON(d.Droplets, out)
}

type DropletAutoscaleHistoryEvents struct {
	History []*godo.DropletAutoscaleHistoryEvent `json:"history"`
}

var _ Displayable = &DropletAutoscaleHistoryEvents{}

func (d *DropletAutoscaleHistoryEvents) Cols() []string {
	return []string{
		"ID",
		"Current Instance",
		"Target Instance",
		"Status",
		"Reason",
		"Error Reason",
	}
}

func (d *DropletAutoscaleHistoryEvents) ColMap() map[string]string {
	return map[string]string{
		"ID":               "ID",
		"Current Instance": "Current Instance",
		"Target Instance":  "Target Instance",
		"Status":           "Status",
		"Reason":           "Reason",
		"Error Reason":     "Error Reason",
	}
}

func (d *DropletAutoscaleHistoryEvents) KV() []map[string]any {
	out := make([]map[string]any, 0, len(d.History))
	for _, history := range d.History {
		out = append(out, map[string]any{
			"ID":               history.HistoryEventID,
			"Current Instance": history.CurrentInstanceCount,
			"Target Instance":  history.DesiredInstanceCount,
			"Status":           history.Status,
			"Reason":           history.Reason,
			"Error Reason":     history.ErrorReason,
		})
	}
	return out
}

func (d *DropletAutoscaleHistoryEvents) JSON(out io.Writer) error {
	return writeJSON(d.History, out)
}
