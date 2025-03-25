package displayers

import (
	"io"
	"strings"

	"github.com/digitalocean/doctl/do"
)

type PartnerNetworkConnect struct {
	PartnerNetworkConnects do.PartnerNetworkConnects
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

	for _, item := range v.PartnerNetworkConnects {
		pnc := item.PartnerNetworkConnect
		o := map[string]any{
			"ID":                        pnc.ID,
			"Name":                      pnc.Name,
			"State":                     pnc.State,
			"ConnectionBandwidthInMbps": pnc.ConnectionBandwidthInMbps,
			"Region":                    pnc.Region,
			"NaaSProvider":              pnc.NaaSProvider,
			"VPCIDs":                    strings.Join(pnc.VPCIDs, ","),
			"CreatedAt":                 pnc.CreatedAt,
			"BGPLocalASN":               pnc.BGP.LocalASN,
			"BGPLocalRouterIP":          pnc.BGP.LocalRouterIP,
			"BGPPeerASN":                pnc.BGP.PeerASN,
			"BGPPeerRouterIP":           pnc.BGP.PeerRouterIP,
		}
		out = append(out, o)
	}

	return out
}

type PartnerNCRoute struct {
	PartnerNetworkConnectRoutes do.PartnerNetworkConnectRoutes
}

var _ Displayable = &PartnerNCRoute{}

func (v *PartnerNCRoute) JSON(out io.Writer) error {
	return writeJSON(v.PartnerNetworkConnectRoutes, out)
}

func (v *PartnerNCRoute) Cols() []string {
	return []string{
		"ID",
		"Cidr",
	}
}

func (v *PartnerNCRoute) ColMap() map[string]string {
	return map[string]string{
		"ID":   "ID",
		"Cidr": "Cidr",
	}
}

func (v *PartnerNCRoute) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.PartnerNetworkConnectRoutes))

	for _, ia := range v.PartnerNetworkConnectRoutes {
		o := map[string]any{
			"ID":   ia.ID,
			"Cidr": ia.Cidr,
		}
		out = append(out, o)
	}

	return out
}

type PartnerNCRegenerateServiceKey struct {
	RegenerateKey do.PartnerNetworkConnectRegenerateServiceKey
}

var _ Displayable = &PartnerNCRegenerateServiceKey{}

func (v *PartnerNCRegenerateServiceKey) JSON(out io.Writer) error {
	return writeJSON(v.RegenerateKey, out)
}

func (v *PartnerNCRegenerateServiceKey) Cols() []string {
	return []string{}
}

func (v *PartnerNCRegenerateServiceKey) ColMap() map[string]string {
	return map[string]string{}
}

func (v *PartnerNCRegenerateServiceKey) KV() []map[string]any {
	out := make([]map[string]any, 0, 1)

	o := map[string]any{}
	out = append(out, o)
	return out
}

type PartnerNCBgpAuthKey struct {
	Key do.PartnerNetworkConnectBGPAuthKey
}

var _ Displayable = &PartnerNCBgpAuthKey{}

func (v *PartnerNCBgpAuthKey) JSON(out io.Writer) error {
	return writeJSON(v.Key, out)
}

func (v *PartnerNCBgpAuthKey) Cols() []string {
	return []string{"Value"}
}

func (v *PartnerNCBgpAuthKey) ColMap() map[string]string {
	return map[string]string{"Value": "Value"}
}

func (v *PartnerNCBgpAuthKey) KV() []map[string]any {
	out := make([]map[string]any, 0, 1)

	o := map[string]any{
		"Value": v.Key.BgpAuthKey.Value,
	}
	out = append(out, o)
	return out
}

type PartnerNCServiceKey struct {
	Key do.PartnerNetworkConnectServiceKey
}

var _ Displayable = &PartnerNCServiceKey{}

func (v *PartnerNCServiceKey) JSON(out io.Writer) error {
	return writeJSON(v.Key, out)
}

func (v *PartnerNCServiceKey) Cols() []string {
	return []string{
		"Value",
		"State",
		"CreatedAt",
	}
}

func (v *PartnerNCServiceKey) ColMap() map[string]string {
	return map[string]string{
		"Value":     "Value",
		"State":     "State",
		"CreatedAt": "CreatedAt",
	}
}

func (v *PartnerNCServiceKey) KV() []map[string]any {
	out := make([]map[string]any, 0, 1)

	o := map[string]any{
		"Value":     v.Key.ServiceKey.Value,
		"State":     v.Key.ServiceKey.State,
		"CreatedAt": v.Key.ServiceKey.CreatedAt,
	}
	out = append(out, o)

	return out
}
