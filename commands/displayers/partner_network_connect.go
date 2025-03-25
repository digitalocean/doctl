package displayers

import (
	"io"
	"strings"

	"github.com/digitalocean/doctl/do"
)

type PartnerNetworkConnect struct {
	PartnerNetworkConnects do.PartnerAttachments
}

var _ Displayable = &PartnerNetworkConnect{}

func (v *PartnerNetworkConnect) JSON(out io.Writer) error {
	return writeJSON(v.PartnerNetworkConnects, out)
}

func (v *PartnerNetworkConnect) Cols() []string {
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

func (v *PartnerNetworkConnect) ColMap() map[string]string {
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

func (v *PartnerNetworkConnect) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.PartnerNetworkConnects))

	for _, ia := range v.PartnerNetworkConnects {
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

type PartnerAttachmentRoute struct {
	PartnerAttachmentRoutes do.PartnerAttachmentRoutes
}

var _ Displayable = &PartnerAttachmentRoute{}

func (v *PartnerAttachmentRoute) JSON(out io.Writer) error {
	return writeJSON(v.PartnerAttachmentRoutes, out)
}

func (v *PartnerAttachmentRoute) Cols() []string {
	return []string{
		"ID",
		"Cidr",
	}
}

func (v *PartnerAttachmentRoute) ColMap() map[string]string {
	return map[string]string{
		"ID":   "ID",
		"Cidr": "Cidr",
	}
}

func (v *PartnerAttachmentRoute) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.PartnerAttachmentRoutes))

	for _, ia := range v.PartnerAttachmentRoutes {
		o := map[string]any{
			"ID":   ia.ID,
			"Cidr": ia.Cidr,
		}
		out = append(out, o)
	}

	return out
}

type PartnerAttachmentRegenerateServiceKey struct {
	RegenerateKey do.PartnerAttachmentRegenerateServiceKey
}

var _ Displayable = &PartnerAttachmentRegenerateServiceKey{}

func (v *PartnerAttachmentRegenerateServiceKey) JSON(out io.Writer) error {
	return writeJSON(v.RegenerateKey, out)
}

func (v *PartnerAttachmentRegenerateServiceKey) Cols() []string {
	return []string{}
}

func (v *PartnerAttachmentRegenerateServiceKey) ColMap() map[string]string {
	return map[string]string{}
}

func (v *PartnerAttachmentRegenerateServiceKey) KV() []map[string]any {
	out := make([]map[string]any, 0, 1)

	o := map[string]any{}
	out = append(out, o)
	return out
}

type PartnerAttachmentBgpAuthKey struct {
	Key do.PartnerAttachmentBGPAuthKey
}

var _ Displayable = &PartnerAttachmentBgpAuthKey{}

func (v *PartnerAttachmentBgpAuthKey) JSON(out io.Writer) error {
	return writeJSON(v.Key, out)
}

func (v *PartnerAttachmentBgpAuthKey) Cols() []string {
	return []string{"Value"}
}

func (v *PartnerAttachmentBgpAuthKey) ColMap() map[string]string {
	return map[string]string{"Value": "Value"}
}

func (v *PartnerAttachmentBgpAuthKey) KV() []map[string]any {
	out := make([]map[string]any, 0, 1)

	o := map[string]any{
		"Value": v.Key.BgpAuthKey.Value,
	}
	out = append(out, o)
	return out
}

type PartnerAttachmentServiceKey struct {
	Key do.PartnerAttachmentServiceKey
}

var _ Displayable = &PartnerAttachmentServiceKey{}

func (v *PartnerAttachmentServiceKey) JSON(out io.Writer) error {
	return writeJSON(v.Key, out)
}

func (v *PartnerAttachmentServiceKey) Cols() []string {
	return []string{
		"Value",
		"State",
		"CreatedAt",
	}
}

func (v *PartnerAttachmentServiceKey) ColMap() map[string]string {
	return map[string]string{
		"Value":     "Value",
		"State":     "State",
		"CreatedAt": "CreatedAt",
	}
}

func (v *PartnerAttachmentServiceKey) KV() []map[string]any {
	out := make([]map[string]any, 0, 1)

	o := map[string]any{
		"Value":     v.Key.ServiceKey.Value,
		"State":     v.Key.ServiceKey.State,
		"CreatedAt": v.Key.ServiceKey.CreatedAt,
	}
	out = append(out, o)

	return out
}
