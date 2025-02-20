package displayers

import (
	"github.com/digitalocean/doctl/do"
	"io"
	"strings"
)

type VPCPeering struct {
	VPCPeerings do.VPCPeerings
}

var _ Displayable = &VPCPeering{}

func (v *VPCPeering) JSON(out io.Writer) error {
	return writeJSON(v.VPCPeerings, out)
}

func (v *VPCPeering) Cols() []string {
	return []string{
		"ID",
		"Name",
		"VPCIDs",
		"Status",
		"Created",
	}
}

func (v *VPCPeering) ColMap() map[string]string {
	return map[string]string{
		"ID":      "ID",
		"Name":    "Name",
		"VPCIDs":  "VPCIDs",
		"Status":  "Status",
		"Created": "Created At",
	}
}

func (v *VPCPeering) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.VPCPeerings))

	for _, v := range v.VPCPeerings {
		o := map[string]any{
			"ID":      v.ID,
			"Name":    v.Name,
			"VPCIDs":  strings.Join(v.VPCIDs, ","),
			"Status":  v.Status,
			"Created": v.CreatedAt,
		}
		out = append(out, o)
	}

	return out
}
