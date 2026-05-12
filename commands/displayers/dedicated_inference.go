package displayers

import (
	"fmt"
	"io"
	"strings"

	"github.com/digitalocean/doctl/do"
)

// DedicatedInference wraps a slice of dedicated inference endpoints for display.
type DedicatedInference struct {
	DedicatedInferences do.DedicatedInferences
}

var _ Displayable = &DedicatedInference{}

func (d *DedicatedInference) JSON(out io.Writer) error {
	return writeJSON(d.DedicatedInferences, out)
}

func (d *DedicatedInference) Cols() []string {
	return []string{
		"ID",
		"Name",
		"Region",
		"Status",
		"VPCUUID",
		"PublicEndpoint",
		"PrivateEndpoint",
		"CreatedAt",
		"UpdatedAt",
	}
}

func (d *DedicatedInference) ColMap() map[string]string {
	return map[string]string{
		"ID":              "ID",
		"Name":            "Name",
		"Region":          "Region",
		"Status":          "Status",
		"VPCUUID":         "VPC UUID",
		"PublicEndpoint":  "Public Endpoint",
		"PrivateEndpoint": "Private Endpoint",
		"CreatedAt":       "Created At",
		"UpdatedAt":       "Updated At",
	}
}

func (d *DedicatedInference) KV() []map[string]any {
	if d == nil || d.DedicatedInferences == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(d.DedicatedInferences))
	for _, di := range d.DedicatedInferences {
		publicEndpoint := ""
		privateEndpoint := ""
		if di.Endpoints != nil {
			publicEndpoint = di.Endpoints.PublicEndpointFQDN
			privateEndpoint = di.Endpoints.PrivateEndpointFQDN
		}
		out = append(out, map[string]any{
			"ID":              di.ID,
			"Name":            di.Name,
			"Region":          di.Region,
			"Status":          di.Status,
			"VPCUUID":         di.VPCUUID,
			"PublicEndpoint":  publicEndpoint,
			"PrivateEndpoint": privateEndpoint,
			"CreatedAt":       di.CreatedAt,
			"UpdatedAt":       di.UpdatedAt,
		})
	}
	return out
}

// DedicatedInferenceAccelerator wraps a slice of accelerator info for display.
type DedicatedInferenceAccelerator struct {
	DedicatedInferenceAcceleratorInfos do.DedicatedInferenceAcceleratorInfos
}

var _ Displayable = &DedicatedInferenceAccelerator{}

func (d *DedicatedInferenceAccelerator) JSON(out io.Writer) error {
	return writeJSON(d.DedicatedInferenceAcceleratorInfos, out)
}

func (d *DedicatedInferenceAccelerator) Cols() []string {
	return []string{
		"ID",
		"Name",
		"Slug",
		"Status",
		"CreatedAt",
	}
}

func (d *DedicatedInferenceAccelerator) ColMap() map[string]string {
	return map[string]string{
		"ID":        "ID",
		"Name":      "Name",
		"Slug":      "Slug",
		"Status":    "Status",
		"CreatedAt": "Created At",
	}
}

func (d *DedicatedInferenceAccelerator) KV() []map[string]any {
	if d == nil || d.DedicatedInferenceAcceleratorInfos == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(d.DedicatedInferenceAcceleratorInfos))
	for _, a := range d.DedicatedInferenceAcceleratorInfos {
		out = append(out, map[string]any{
			"ID":        a.ID,
			"Name":      a.Name,
			"Slug":      a.Slug,
			"Status":    a.Status,
			"CreatedAt": a.CreatedAt,
		})
	}
	return out
}

// DedicatedInferenceList wraps a slice of dedicated inference list items for display.
type DedicatedInferenceList struct {
	DedicatedInferenceListItems do.DedicatedInferenceListItems
}

var _ Displayable = &DedicatedInferenceList{}

func (d *DedicatedInferenceList) JSON(out io.Writer) error {
	return writeJSON(d.DedicatedInferenceListItems, out)
}

func (d *DedicatedInferenceList) Cols() []string {
	return []string{
		"ID",
		"Name",
		"Region",
		"Status",
		"VPCUUID",
		"PublicEndpoint",
		"PrivateEndpoint",
		"CreatedAt",
		"UpdatedAt",
	}
}

func (d *DedicatedInferenceList) ColMap() map[string]string {
	return map[string]string{
		"ID":              "ID",
		"Name":            "Name",
		"Region":          "Region",
		"Status":          "Status",
		"VPCUUID":         "VPC UUID",
		"PublicEndpoint":  "Public Endpoint",
		"PrivateEndpoint": "Private Endpoint",
		"CreatedAt":       "Created At",
		"UpdatedAt":       "Updated At",
	}
}

func (d *DedicatedInferenceList) KV() []map[string]any {
	if d == nil || d.DedicatedInferenceListItems == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(d.DedicatedInferenceListItems))
	for _, di := range d.DedicatedInferenceListItems {
		publicEndpoint := ""
		privateEndpoint := ""
		if di.Endpoints != nil {
			publicEndpoint = di.Endpoints.PublicEndpointFQDN
			privateEndpoint = di.Endpoints.PrivateEndpointFQDN
		}
		out = append(out, map[string]any{
			"ID":              di.ID,
			"Name":            di.Name,
			"Region":          di.Region,
			"Status":          di.Status,
			"VPCUUID":         di.VPCUUID,
			"PublicEndpoint":  publicEndpoint,
			"PrivateEndpoint": privateEndpoint,
			"CreatedAt":       di.CreatedAt,
			"UpdatedAt":       di.UpdatedAt,
		})
	}
	return out
}

// DedicatedInferenceTokenDisplayer wraps a slice of dedicated inference tokens for display.
type DedicatedInferenceTokenDisplayer struct {
	DedicatedInferenceTokens []do.DedicatedInferenceToken
}

var _ Displayable = &DedicatedInferenceTokenDisplayer{}

func (d *DedicatedInferenceTokenDisplayer) JSON(out io.Writer) error {
	return writeJSON(d.DedicatedInferenceTokens, out)
}

func (d *DedicatedInferenceTokenDisplayer) Cols() []string {
	return []string{
		"ID",
		"Name",
		"IsManaged",
		"Value",
		"CreatedAt",
	}
}

func (d *DedicatedInferenceTokenDisplayer) ColMap() map[string]string {
	return map[string]string{
		"ID":        "ID",
		"Name":      "Name",
		"IsManaged": "Is Managed",
		"Value":     "Value",
		"CreatedAt": "Created At",
	}
}

func (d *DedicatedInferenceTokenDisplayer) KV() []map[string]any {
	if d == nil || d.DedicatedInferenceTokens == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(d.DedicatedInferenceTokens))
	for _, t := range d.DedicatedInferenceTokens {
		out = append(out, map[string]any{
			"ID":        t.ID,
			"Name":      t.Name,
			"IsManaged": t.IsManaged,
			"Value":     t.Value,
			"CreatedAt": t.CreatedAt,
		})
	}
	return out
}

// DedicatedInferenceSizeDisplayer wraps a slice of dedicated inference sizes for display.
type DedicatedInferenceSizeDisplayer struct {
	DedicatedInferenceSizes do.DedicatedInferenceSizes
}

var _ Displayable = &DedicatedInferenceSizeDisplayer{}

func (d *DedicatedInferenceSizeDisplayer) JSON(out io.Writer) error {
	return writeJSON(d.DedicatedInferenceSizes, out)
}

func (d *DedicatedInferenceSizeDisplayer) Cols() []string {
	return []string{
		"GPUSlug",
		"PricePerHour",
		"Currency",
		"CPU",
		"Memory",
		"GPUCount",
		"GPUVramGB",
		"GPUModel",
		"Regions",
	}
}

func (d *DedicatedInferenceSizeDisplayer) ColMap() map[string]string {
	return map[string]string{
		"GPUSlug":      "GPU Slug",
		"PricePerHour": "Price/Hour",
		"Currency":     "Currency",
		"CPU":          "CPU",
		"Memory":       "Memory (MB)",
		"GPUCount":     "GPU Count",
		"GPUVramGB":    "GPU VRAM (GB)",
		"GPUModel":     "GPU Model",
		"Regions":      "Regions",
	}
}

func (d *DedicatedInferenceSizeDisplayer) KV() []map[string]any {
	if d == nil || d.DedicatedInferenceSizes == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(d.DedicatedInferenceSizes))
	for _, sz := range d.DedicatedInferenceSizes {
		gpuCount := uint32(0)
		gpuVramGB := uint32(0)
		gpuModel := ""
		if sz.GPU != nil {
			gpuCount = sz.GPU.Count
			gpuVramGB = sz.GPU.VramGb
			gpuModel = sz.GPU.Slug
		}
		out = append(out, map[string]any{
			"GPUSlug":      sz.GPUSlug,
			"PricePerHour": fmt.Sprintf("%s %s", sz.PricePerHour, sz.Currency),
			"Currency":     sz.Currency,
			"CPU":          sz.CPU,
			"Memory":       sz.Memory,
			"GPUCount":     gpuCount,
			"GPUVramGB":    gpuVramGB,
			"GPUModel":     gpuModel,
			"Regions":      strings.Join(sz.Regions, ","),
		})
	}
	return out
}

// DedicatedInferenceGPUModelConfigDisplayer wraps a slice of GPU model configs for display.
type DedicatedInferenceGPUModelConfigDisplayer struct {
	DedicatedInferenceGPUModelConfigs do.DedicatedInferenceGPUModelConfigs
}

var _ Displayable = &DedicatedInferenceGPUModelConfigDisplayer{}

func (d *DedicatedInferenceGPUModelConfigDisplayer) JSON(out io.Writer) error {
	return writeJSON(d.DedicatedInferenceGPUModelConfigs, out)
}

func (d *DedicatedInferenceGPUModelConfigDisplayer) Cols() []string {
	return []string{
		"ModelSlug",
		"ModelName",
		"IsModelGated",
		"GPUSlugs",
	}
}

func (d *DedicatedInferenceGPUModelConfigDisplayer) ColMap() map[string]string {
	return map[string]string{
		"ModelSlug":    "Model Slug",
		"ModelName":    "Model Name",
		"IsModelGated": "Gated",
		"GPUSlugs":     "GPU Slugs",
	}
}

func (d *DedicatedInferenceGPUModelConfigDisplayer) KV() []map[string]any {
	if d == nil || d.DedicatedInferenceGPUModelConfigs == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(d.DedicatedInferenceGPUModelConfigs))
	for _, cfg := range d.DedicatedInferenceGPUModelConfigs {
		out = append(out, map[string]any{
			"ModelSlug":    cfg.ModelSlug,
			"ModelName":    cfg.ModelName,
			"IsModelGated": cfg.IsModelGated,
			"GPUSlugs":     strings.Join(cfg.GPUSlugs, ","),
		})
	}
	return out
}
