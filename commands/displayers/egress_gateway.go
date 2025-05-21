package displayers

import (
	"fmt"
	"io"
	"strings"

	"github.com/digitalocean/godo"
)

type EgressGateways struct {
	EgressGateways []*godo.EgressGateway `json:"egress_gateways"`
}

var _ Displayable = &EgressGateways{}

func (e *EgressGateways) Cols() []string {
	return []string{
		"ID",
		"Name",
		"Type",
		"State",
		"Region",
		"VPCs",
		"Egresses",
		"Timeouts",
	}
}

func (e *EgressGateways) ColMap() map[string]string {
	return map[string]string{
		"ID":       "ID",
		"Name":     "Name",
		"Type":     "Type",
		"State":    "State",
		"Region":   "Region",
		"VPCs":     "VPCs",
		"Egresses": "Egresses",
		"Timeouts": "Timeouts",
	}
}

func (e *EgressGateways) KV() []map[string]any {
	out := make([]map[string]any, 0, len(e.EgressGateways))
	for _, gateway := range e.EgressGateways {
		out = append(out, map[string]any{
			"ID":     gateway.ID,
			"Name":   gateway.Name,
			"Type":   gateway.Type,
			"State":  gateway.State,
			"Region": gateway.Region,
			"VPCs": func() string {
				var vpcs []string
				for _, vpc := range gateway.VPCs {
					vpcs = append(vpcs, fmt.Sprintf("%s:%s", vpc.VpcUUID, vpc.GatewayIP))
				}
				return strings.Join(vpcs, ",")
			}(),
			"Egresses": func() string {
				var egresses []string
				if gateway.Egresses != nil {
					for _, egress := range gateway.Egresses.PublicGateways {
						egresses = append(egresses, egress.IPv4)
					}
				}
				return strings.Join(egresses, ",")
			}(),
			"Timeouts": func() string {
				return fmt.Sprintf("udp:%ds,icmp:%ds,tcp:%ds",
					gateway.UDPTimeoutSeconds,
					gateway.ICMPTimeoutSeconds,
					gateway.TCPTimeoutSeconds)
			}(),
		})
	}
	return out
}

func (e *EgressGateways) JSON(out io.Writer) error {
	return writeJSON(e.EgressGateways, out)
}
