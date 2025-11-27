package displayers

import (
	"fmt"
	"io"
	"strings"

	"github.com/digitalocean/godo"
)

type VPCNATGateways struct {
	VPCNATGateways []*godo.VPCNATGateway `json:"vpc_nat_gateways"`
}

var _ Displayable = &VPCNATGateways{}

func (e *VPCNATGateways) Cols() []string {
	return []string{
		"ID",
		"Name",
		"Type",
		"State",
		"Region",
		"VPC",
		"GatewayIP",
		"Default",
		"Egresses",
		"Timeouts",
	}
}

func (e *VPCNATGateways) ColMap() map[string]string {
	return map[string]string{
		"ID":        "ID",
		"Name":      "Name",
		"Type":      "Type",
		"State":     "State",
		"Region":    "Region",
		"VPC":       "VPC",
		"GatewayIP": "GatewayIP",
		"Default":   "Default",
		"Egresses":  "Egresses",
		"Timeouts":  "Timeouts",
	}
}

func (e *VPCNATGateways) KV() []map[string]any {
	out := make([]map[string]any, 0, len(e.VPCNATGateways))
	for _, gateway := range e.VPCNATGateways {
		for id, vpc := range gateway.VPCs {
			var (
				rowGw  godo.VPCNATGateway
				rowMap = make(map[string]any)
			)
			if id == 0 {
				rowGw = *gateway
			}
			rowMap["ID"] = rowGw.ID
			rowMap["Name"] = rowGw.Name
			rowMap["Type"] = rowGw.Type
			rowMap["State"] = rowGw.State
			rowMap["Region"] = rowGw.Region
			rowMap["VPC"] = vpc.VpcUUID
			rowMap["GatewayIP"] = vpc.GatewayIP
			rowMap["Default"] = vpc.DefaultGateway
			rowMap["Egresses"] = func() string {
				var egresses []string
				if rowGw.Egresses != nil {
					for _, egress := range rowGw.Egresses.PublicGateways {
						egresses = append(egresses, egress.IPv4)
					}
				}
				return strings.Join(egresses, ",")
			}()
			rowMap["Timeouts"] = func() string {
				if rowGw.UDPTimeoutSeconds > 0 && rowGw.ICMPTimeoutSeconds > 0 && rowGw.TCPTimeoutSeconds > 0 {
					return fmt.Sprintf("udp:%ds,icmp:%ds,tcp:%ds",
						rowGw.UDPTimeoutSeconds,
						rowGw.ICMPTimeoutSeconds,
						rowGw.TCPTimeoutSeconds)
				}
				return ""
			}()
			out = append(out, rowMap)
		}
	}
	return out
}

func (e *VPCNATGateways) JSON(out io.Writer) error {
	return writeJSON(e.VPCNATGateways, out)
}
