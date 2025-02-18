package displayers

import (
	"io"
	"strings"

	"github.com/digitalocean/doctl/do"
)

type PartnerInterconnectAttachment struct {
	PartnerInterconnectAttachments do.PartnerInterconnectAttachments
}

var _ Displayable = &PartnerInterconnectAttachment{}

func (v *PartnerInterconnectAttachment) JSON(out io.Writer) error {
	return writeJSON(v.PartnerInterconnectAttachments, out)
}

func (v *PartnerInterconnectAttachment) Cols() []string {
	return []string{
		"ID",
		"Name",
		"State",
		"ConnectionBandwidthInMbps",
		"Region",
		"NaaSProvider",
		"VPCIDs",
		"CreatedAt",
		"BGPLocalASN",
		"BGPLocalRouterIP",
		"BGPPeerASN",
		"BGPPeerRouterIP",
	}
}

func (v *PartnerInterconnectAttachment) ColMap() map[string]string {
	return map[string]string{
		"ID":                        "ID",
		"Name":                      "Name",
		"State":                     "State",
		"ConnectionBandwidthInMbps": "Connection Bandwidth (MBPS)",
		"Region":                    "Region",
		"NaaSProvider":              "NaaS Provider",
		"VPCIDs":                    "VPC IDs",
		"CreatedAt":                 "Created At",
		"BGPLocalASN":               "BGP Local ASN",
		"BGPLocalRouterIP":          "BGP Local Router IP",
		"BGPPeerASN":                "BGP Peer ASN",
		"BGPPeerRouterIP":           "BGP Peer Router IP",
	}
}

func (v *PartnerInterconnectAttachment) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.PartnerInterconnectAttachments))

	for _, ia := range v.PartnerInterconnectAttachments {
		o := map[string]any{
			"ID":                        ia.ID,
			"Name":                      ia.Name,
			"State":                     ia.State,
			"ConnectionBandwidthInMbps": ia.ConnectionBandwidthInMbps,
			"Region":                    ia.Region,
			"NaaSProvider":              ia.NaaSProvider,
			"VPCIDs":                    strings.Join(ia.VPCIDs, ","),
			"CreatedAt":                 ia.CreatedAt,
			"BGPLocalASN":               ia.BGP.LocalASN,
			"BGPLocalRouterIP":          ia.BGP.LocalRouterIP,
			"BGPPeerASN":                ia.BGP.PeerASN,
			"BGPPeerRouterIP":           ia.BGP.PeerRouterIP,
		}
		out = append(out, o)
	}

	return out
}

type PartnerInterconnectAttachmentServiceKey struct {
	Key do.PartnerInterconnectAttachmentServiceKey
}

var _ Displayable = &PartnerInterconnectAttachmentServiceKey{}

func (v *PartnerInterconnectAttachmentServiceKey) JSON(out io.Writer) error {
	return writeJSON(v.Key, out)
}

func (v *PartnerInterconnectAttachmentServiceKey) Cols() []string {
	return []string{
		"Key",
		"State",
	}
}

func (v *PartnerInterconnectAttachmentServiceKey) ColMap() map[string]string {
	return map[string]string{
		"Key":   "Key",
		"State": "State",
	}
}

func (v *PartnerInterconnectAttachmentServiceKey) KV() []map[string]any {
	out := make([]map[string]any, 0, 1)

	o := map[string]any{
		"Key":   v.Key.ServiceKey,
		"State": v.Key.State,
	}
	out = append(out, o)

	return out
}
