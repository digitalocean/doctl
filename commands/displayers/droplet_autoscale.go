package displayers

import (
	"io"

	"github.com/digitalocean/godo"
)

type DropletAutoscale struct {
	AutoscalePools []*godo.DropletAutoscalePool         `json:"autoscale_pools"`
	Droplets       []*godo.DropletAutoscaleResource     `json:"droplets"`
	History        []*godo.DropletAutoscaleHistoryEvent `json:"history"`
}

var _ Displayable = &DropletAutoscale{}

func (d *DropletAutoscale) JSON(out io.Writer) error {
	switch {
	case d.AutoscalePools != nil:
		return writeJSON(d.AutoscalePools, out)
	case d.Droplets != nil:
		return writeJSON(d.Droplets, out)
	case d.History != nil:
		return writeJSON(d.History, out)
	}
	return nil
}

func (d *DropletAutoscale) Cols() []string {
	switch {
	case d.AutoscalePools != nil:
		return []string{
			"ID",
			"NAME",
			"REGION",
			"STATUS",
			"MIN INSTANCE",
			"MAX INSTANCE",
			"TARGET INSTANCE",
			"AVG CPU UTIL",
			"AVG MEM UTIL",
			"TARGET CPU UTIL",
			"TARGET MEM UTIL",
		}
	case d.Droplets != nil:
		return []string{
			"ID",
			"STATUS",
			"HEALTH STATUS",
			"UNHEALTHY REASON",
			"CPU UTIL",
			"MEM UTIL",
		}
	case d.History != nil:
		return []string{
			"ID",
			"CURRENT INSTANCE",
			"TARGET INSTANCE",
			"STATUS",
			"REASON",
			"ERROR REASON",
		}
	}
	return nil
}

func (d *DropletAutoscale) ColMap() map[string]string {
	switch {
	case d.AutoscalePools != nil:
		return map[string]string{
			"ID":              "ID",
			"NAME":            "NAME",
			"REGION":          "REGION",
			"STATUS":          "STATUS",
			"MIN INSTANCE":    "MIN INSTANCE",
			"MAX INSTANCE":    "MAX INSTANCE",
			"TARGET INSTANCE": "TARGET INSTANCE",
			"AVG CPU UTIL":    "AVG CPU UTIL",
			"AVG MEM UTIL":    "AVG MEM UTIL",
			"TARGET CPU UTIL": "TARGET CPU UTIL",
			"TARGET MEM UTIL": "TARGET MEM UTIL",
		}
	case d.Droplets != nil:
		return map[string]string{
			"ID":               "ID",
			"STATUS":           "STATUS",
			"HEALTH STATUS":    "HEALTH STATUS",
			"UNHEALTHY REASON": "UNHEALTHY REASON",
			"CPU UTIL":         "CPU UTIL",
			"MEM UTIL":         "MEM UTIL",
		}
	case d.History != nil:
		return map[string]string{
			"ID":               "ID",
			"CURRENT INSTANCE": "CURRENT INSTANCE",
			"TARGET INSTANCE":  "TARGET INSTANCE",
			"STATUS":           "STATUS",
			"REASON":           "REASON",
			"ERROR REASON":     "ERROR REASON",
		}
	}
	return nil
}

func (d *DropletAutoscale) KV() []map[string]any {
	switch {
	case d.AutoscalePools != nil:
		out := make([]map[string]any, 0, len(d.AutoscalePools))
		for _, pool := range d.AutoscalePools {
			var cpuUtil, memUtil any
			if pool.CurrentUtilization != nil {
				cpuUtil = pool.CurrentUtilization.CPU
				memUtil = pool.CurrentUtilization.Memory
			}
			out = append(out, map[string]any{
				"ID":              pool.ID,
				"NAME":            pool.Name,
				"REGION":          pool.DropletTemplate.Region,
				"STATUS":          pool.Status,
				"MIN INSTANCE":    pool.Config.MinInstances,
				"MAX INSTANCE":    pool.Config.MaxInstances,
				"TARGET INSTANCE": pool.Config.TargetNumberInstances,
				"AVG CPU UTIL":    cpuUtil,
				"AVG MEM UTIL":    memUtil,
				"TARGET CPU UTIL": pool.Config.TargetCPUUtilization,
				"TARGET MEM UTIL": pool.Config.TargetCPUUtilization,
			})
		}
		return out
	case d.Droplets != nil:
		out := make([]map[string]any, 0, len(d.Droplets))
		for _, droplet := range d.Droplets {
			var cpuUtil, memUtil any
			if droplet.CurrentUtilization != nil {
				cpuUtil = droplet.CurrentUtilization.CPU
				memUtil = droplet.CurrentUtilization.Memory
			}
			out = append(out, map[string]any{
				"ID":               droplet.DropletID,
				"STATUS":           droplet.Status,
				"HEALTH STATUS":    droplet.HealthStatus,
				"UNHEALTHY REASON": droplet.UnhealthyReason,
				"CPU UTIL":         cpuUtil,
				"MEM UTIL":         memUtil,
			})
		}
		return out
	case d.History != nil:
		out := make([]map[string]any, 0, len(d.History))
		for _, history := range d.History {
			out = append(out, map[string]any{
				"ID":               history.HistoryEventID,
				"CURRENT INSTANCE": history.CurrentInstanceCount,
				"TARGET INSTANCE":  history.DesiredInstanceCount,
				"STATUS":           history.Status,
				"REASON":           history.Reason,
				"ERROR REASON":     history.ErrorReason,
			})
		}
		return out
	}
	return nil
}
