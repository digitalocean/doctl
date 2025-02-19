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

type PartnerInterconnectAttachmentRoute struct {
	PartnerInterconnectAttachmentRoutes do.PartnerInterconnectAttachmentRoutes
}

var _ Displayable = &PartnerInterconnectAttachmentRoute{}

func (v *PartnerInterconnectAttachmentRoute) JSON(out io.Writer) error {
	return writeJSON(v.PartnerInterconnectAttachmentRoutes, out)
}

func (v *PartnerInterconnectAttachmentRoute) Cols() []string {
	return []string{
		"ID",
		"Cidr",
	}
}

func (v *PartnerInterconnectAttachmentRoute) ColMap() map[string]string {
	return map[string]string{
		"ID":   "ID",
		"Cidr": "Cidr",
	}
}

func (v *PartnerInterconnectAttachmentRoute) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.PartnerInterconnectAttachmentRoutes))

	for _, ia := range v.PartnerInterconnectAttachmentRoutes {
		o := map[string]any{
			"ID":   ia.ID,
			"Cidr": ia.Cidr,
		}
		out = append(out, o)
	}

	return out
}

type PartnerInterconnectAttachmentRegenerateServiceKey struct {
	RegenerateKey do.PartnerInterconnectAttachmentRegenerateServiceKey
}

var _ Displayable = &PartnerInterconnectAttachmentRegenerateServiceKey{}

func (v *PartnerInterconnectAttachmentRegenerateServiceKey) JSON(out io.Writer) error {
	return writeJSON(v.RegenerateKey, out)
}

func (v *PartnerInterconnectAttachmentRegenerateServiceKey) Cols() []string {
	return []string{}
}

func (v *PartnerInterconnectAttachmentRegenerateServiceKey) ColMap() map[string]string {
	return map[string]string{}
}

func (v *PartnerInterconnectAttachmentRegenerateServiceKey) KV() []map[string]any {
	out := make([]map[string]any, 0, 1)

	o := map[string]any{}
	out = append(out, o)
	return out
}

type PartnerInterconnectAttachmentBgpAuthKey struct {
	Key do.PartnerInterconnectAttachmentBGPAuthKey
}

var _ Displayable = &PartnerInterconnectAttachmentBgpAuthKey{}

func (v *PartnerInterconnectAttachmentBgpAuthKey) JSON(out io.Writer) error {
	return writeJSON(v.Key, out)
}

func (v *PartnerInterconnectAttachmentBgpAuthKey) Cols() []string {
	return []string{"Value"}
}

func (v *PartnerInterconnectAttachmentBgpAuthKey) ColMap() map[string]string {
	return map[string]string{"Value": "Value"}
}

func (v *PartnerInterconnectAttachmentBgpAuthKey) KV() []map[string]any {
	out := make([]map[string]any, 0, 1)

	o := map[string]any{
		"Value": v.Key.BgpAuthKey.Value,
	}
	out = append(out, o)
	return out
}
