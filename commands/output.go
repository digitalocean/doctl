/*
Copyright 2016 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/digitalocean/doctl/do"
)

var (
	hc = &headerControl{}
)

func newTabWriter(out io.Writer) *tabwriter.Writer {
	w := new(tabwriter.Writer)
	w.Init(out, 0, 0, 4, ' ', 0)

	return w
}

type headerControl struct {
	hideHeader bool
}

func (hc *headerControl) HideHeader(hide bool) {
	hc.hideHeader = hide
}

type rateLimit struct {
	*do.RateLimit
}

var _ Displayable = &rateLimit{}

func (rl *rateLimit) JSON(out io.Writer) error {
	return writeJSON(rl.Rate, out)
}

func (rl *rateLimit) Cols() []string {
	return []string{
		"Limit", "Remaining", "Reset",
	}
}

func (rl *rateLimit) ColMap() map[string]string {
	return map[string]string{
		"Limit": "Limit", "Remaining": "Remaining", "Reset": "Reset",
	}
}

func (rl *rateLimit) KV() []map[string]interface{} {
	out := []map[string]interface{}{}
	x := map[string]interface{}{
		"Limit": rl.Limit, "Remaining": rl.Remaining, "Reset": rl.Reset,
	}
	out = append(out, x)

	return out
}

type account struct {
	*do.Account
}

var _ Displayable = &account{}

func (a *account) JSON(out io.Writer) error {
	return writeJSON(a.Account, out)
}

func (a *account) Cols() []string {
	return []string{
		"Email", "DropletLimit", "EmailVerified", "UUID", "Status",
	}
}

func (a *account) ColMap() map[string]string {
	return map[string]string{
		"Email": "Email", "DropletLimit": "Droplet Limit", "EmailVerified": "Email Verified",
		"UUID": "UUID", "Status": "Status",
	}
}

func (a *account) KV() []map[string]interface{} {
	out := []map[string]interface{}{}
	x := map[string]interface{}{
		"Email": a.Email, "DropletLimit": a.DropletLimit,
		"EmailVerified": a.EmailVerified, "UUID": a.UUID,
		"Status": a.Status,
	}
	out = append(out, x)

	return out
}

type action struct {
	actions do.Actions
}

var _ Displayable = &action{}

func (a *action) JSON(out io.Writer) error {
	return writeJSON(a.actions, out)
}

func (a *action) Cols() []string {
	return []string{
		"ID", "Status", "Type", "StartedAt", "CompletedAt", "ResourceID", "ResourceType", "Region",
	}
}

func (a *action) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Status": "Status", "Type": "Type", "StartedAt": "Started At",
		"CompletedAt": "Completed At", "ResourceID": "Resource ID",
		"ResourceType": "Resource Type", "Region": "Region",
	}
}

func (a *action) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, x := range a.actions {
		region := ""
		if x.Region != nil {
			region = x.Region.Slug
		}
		o := map[string]interface{}{
			"ID": x.ID, "Status": x.Status, "Type": x.Type,
			"StartedAt": x.StartedAt, "CompletedAt": x.CompletedAt,
			"ResourceID": x.ResourceID, "ResourceType": x.ResourceType,
			"Region": region,
		}
		out = append(out, o)
	}

	return out
}

type domain struct {
	domains do.Domains
}

var _ Displayable = &domain{}

func (d *domain) JSON(out io.Writer) error {
	return writeJSON(d.domains, out)
}

func (d *domain) Cols() []string {
	return []string{"Domain", "TTL"}
}

func (d *domain) ColMap() map[string]string {
	return map[string]string{
		"Domain": "Domain", "TTL": "TTL",
	}
}

func (d *domain) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, do := range d.domains {
		o := map[string]interface{}{
			"Domain": do.Name, "TTL": do.TTL,
		}
		out = append(out, o)
	}

	return out
}

type domainRecord struct {
	domainRecords do.DomainRecords
}

func (dr *domainRecord) JSON(out io.Writer) error {
	return writeJSON(dr.domainRecords, out)
}

func (dr *domainRecord) Cols() []string {
	return []string{
		"ID", "Type", "Name", "Data", "Priority", "Port", "TTL", "Weight",
	}
}

func (dr *domainRecord) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Type": "Type", "Name": "Name", "Data": "Data",
		"Priority": "Priority", "Port": "Port", "TTL": "TTL", "Weight": "Weight",
	}
}

func (dr *domainRecord) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, d := range dr.domainRecords {
		o := map[string]interface{}{
			"ID": d.ID, "Type": d.Type, "Name": d.Name,
			"Data": d.Data, "Priority": d.Priority,
			"Port": d.Port, "TTL": d.TTL, "Weight": d.Weight,
		}
		out = append(out, o)
	}

	return out
}

type droplet struct {
	droplets do.Droplets
}

var _ Displayable = &droplet{}

func (d *droplet) JSON(out io.Writer) error {
	return writeJSON(d.droplets, out)
}

func (d *droplet) Cols() []string {
	cols := []string{
		"ID", "Name", "PublicIPv4", "PrivateIPv4", "PublicIPv6", "Memory", "VCPUs", "Disk", "Region", "Image", "Status", "Tags", "Features", "Volumes",
	}
	return cols
}

func (d *droplet) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "PublicIPv4": "Public IPv4", "PrivateIPv4": "Private IPv4", "PublicIPv6": "Public IPv6",
		"Memory": "Memory", "VCPUs": "VCPUs", "Disk": "Disk",
		"Region": "Region", "Image": "Image", "Status": "Status",
		"Tags": "Tags", "Features": "Features", "Volumes": "Volumes",
		"SizeSlug": "Size Slug",
	}
}

func (d *droplet) KV() []map[string]interface{} {
	out := []map[string]interface{}{}
	for _, d := range d.droplets {
		tags := strings.Join(d.Tags, ",")
		image := fmt.Sprintf("%s %s", d.Image.Distribution, d.Image.Name)
		ip, _ := d.PublicIPv4()
		privIP, _ := d.PrivateIPv4()
		ip6, _ := d.PublicIPv6()
		features := strings.Join(d.Features, ",")
		volumes := strings.Join(d.VolumeIDs, ",")
		m := map[string]interface{}{
			"ID": d.ID, "Name": d.Name, "PublicIPv4": ip, "PrivateIPv4": privIP, "PublicIPv6": ip6,
			"Memory": d.Memory, "VCPUs": d.Vcpus, "Disk": d.Disk,
			"Region": d.Region.Slug, "Image": image, "Status": d.Status,
			"Tags": tags, "Features": features, "Volumes": volumes,
			"SizeSlug": d.SizeSlug,
		}
		out = append(out, m)
	}

	return out
}

type floatingIP struct {
	floatingIPs do.FloatingIPs
}

var _ Displayable = &floatingIP{}

func (fi *floatingIP) JSON(out io.Writer) error {
	return writeJSON(fi.floatingIPs, out)
}

func (fi *floatingIP) Cols() []string {
	return []string{
		"IP", "Region", "DropletID", "DropletName",
	}
}

func (fi *floatingIP) ColMap() map[string]string {
	return map[string]string{
		"IP": "IP", "Region": "Region", "DropletID": "Droplet ID", "DropletName": "Droplet Name",
	}
}

func (fi *floatingIP) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, f := range fi.floatingIPs {
		var dropletID, dropletName string
		if f.Droplet != nil {
			dropletID = fmt.Sprintf("%d", f.Droplet.ID)
			dropletName = f.Droplet.Name
		}

		o := map[string]interface{}{
			"IP": f.IP, "Region": f.Region.Slug,
			"DropletID": dropletID, "DropletName": dropletName,
		}

		out = append(out, o)
	}

	return out
}

type image struct {
	images do.Images
}

var _ Displayable = &image{}

func (gi *image) JSON(out io.Writer) error {
	return writeJSON(gi.images, out)
}

func (gi *image) Cols() []string {
	return []string{
		"ID", "Name", "Type", "Distribution", "Slug", "Public", "MinDisk",
	}
}

func (gi *image) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "Type": "Type", "Distribution": "Distribution",
		"Slug": "Slug", "Public": "Public", "MinDisk": "Min Disk",
	}
}

func (gi *image) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, i := range gi.images {
		publicStatus := false
		if i.Public {
			publicStatus = true
		}

		o := map[string]interface{}{
			"ID": i.ID, "Name": i.Name, "Type": i.Type, "Distribution": i.Distribution,
			"Slug": i.Slug, "Public": publicStatus, "MinDisk": i.MinDiskSize,
		}

		out = append(out, o)
	}

	return out
}

type kernel struct {
	kernels do.Kernels
}

var _ Displayable = &kernel{}

func (ke *kernel) JSON(out io.Writer) error {
	return writeJSON(ke.kernels, out)
}

func (ke *kernel) Cols() []string {
	return []string{
		"ID", "Name", "Version",
	}
}

func (ke *kernel) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "Version": "Version",
	}
}

func (ke *kernel) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, k := range ke.kernels {
		o := map[string]interface{}{
			"ID": k.ID, "Name": k.Name, "Version": k.Version,
		}

		out = append(out, o)
	}

	return out
}

type key struct {
	keys do.SSHKeys
}

var _ Displayable = &key{}

func (ke *key) JSON(out io.Writer) error {
	return writeJSON(ke.keys, out)
}

func (ke *key) Cols() []string {
	return []string{
		"ID", "Name", "FingerPrint",
	}
}

func (ke *key) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "FingerPrint": "FingerPrint",
	}
}

func (ke *key) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, k := range ke.keys {
		o := map[string]interface{}{
			"ID": k.ID, "Name": k.Name, "FingerPrint": k.Fingerprint,
		}

		out = append(out, o)
	}

	return out
}

type region struct {
	regions do.Regions
}

var _ Displayable = &region{}

func (re *region) JSON(out io.Writer) error {
	return writeJSON(re.regions, out)
}

func (re *region) Cols() []string {
	return []string{
		"Slug", "Name", "Available",
	}
}

func (re *region) ColMap() map[string]string {
	return map[string]string{
		"Slug": "Slug", "Name": "Name", "Available": "Available",
	}
}

func (re *region) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, r := range re.regions {
		o := map[string]interface{}{
			"Slug": r.Slug, "Name": r.Name, "Available": r.Available,
		}

		out = append(out, o)
	}

	return out
}

type size struct {
	sizes do.Sizes
}

var _ Displayable = &size{}

func (si *size) JSON(out io.Writer) error {
	return writeJSON(si.sizes, out)
}

func (si *size) Cols() []string {
	return []string{
		"Slug", "Memory", "VCPUs", "Disk", "PriceMonthly", "PriceHourly",
	}
}

func (si *size) ColMap() map[string]string {
	return map[string]string{
		"Slug": "Slug", "Memory": "Memory", "VCPUs": "VCPUs",
		"Disk": "Disk", "PriceMonthly": "Price Monthly",
		"PriceHourly": "Price Hourly",
	}
}

func (si *size) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, s := range si.sizes {
		o := map[string]interface{}{
			"Slug": s.Slug, "Memory": s.Memory, "VCPUs": s.Vcpus,
			"Disk": s.Disk, "PriceMonthly": fmt.Sprintf("%0.2f", s.PriceMonthly),
			"PriceHourly": s.PriceHourly,
		}

		out = append(out, o)
	}

	return out
}

type plugin struct {
	plugins []plugDesc
}

var _ Displayable = &plugin{}

func (p *plugin) JSON(out io.Writer) error {
	return writeJSON(p.plugins, out)
}

func (p *plugin) Cols() []string {
	return []string{
		"Name",
	}
}

func (p *plugin) ColMap() map[string]string {
	return map[string]string{
		"Name": "Name",
	}
}

func (p *plugin) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, plug := range p.plugins {
		o := map[string]interface{}{
			"Name": plug.Name,
		}

		out = append(out, o)
	}

	return out
}

type tag struct {
	tags do.Tags
}

var _ Displayable = &action{}

func (t *tag) JSON(out io.Writer) error {
	return writeJSON(t.tags, out)
}

func (t *tag) Cols() []string {
	return []string{"Name", "DropletCount"}
}

func (t *tag) ColMap() map[string]string {
	return map[string]string{
		"Name":         "Name",
		"DropletCount": "Droplet Count",
	}
}

func (t *tag) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, x := range t.tags {
		dropletCount := x.Resources.Droplets.Count
		o := map[string]interface{}{
			"Name":         x.Name,
			"DropletCount": dropletCount,
		}
		out = append(out, o)
	}

	return out
}

//CDN Output

type cdn struct {
	cdns []do.CDN
}

var _ Displayable = &cdn{}

func (c *cdn) JSON(out io.Writer) error {
	return writeJSON(c.cdns, out)
}

func (c *cdn) Cols() []string {
	return []string{
		"ID", "Origin", "Endpoint", "TTL", "CreatedAt",
	}
}

func (c *cdn) ColMap() map[string]string {
	return map[string]string{
		"ID":        "ID",
		"Origin":    "Origin",
		"Endpoint":  "Endpoint",
		"TTL":       "TTL",
		"CreatedAt": "CreatedAt",
	}
}

func (c *cdn) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, cdn := range c.cdns {
		m := map[string]interface{}{
			"ID":        cdn.ID,
			"Origin":    cdn.Origin,
			"Endpoint":  cdn.Endpoint,
			"TTL":       cdn.TTL,
			"CreatedAt": cdn.CreatedAt,
		}

		out = append(out, m)
	}

	return out
}

type volume struct {
	volumes []do.Volume
}

var _ Displayable = &volume{}

func (a *volume) JSON(out io.Writer) error {
	return writeJSON(a.volumes, out)

}

func (a *volume) Cols() []string {
	return []string{
		"ID", "Name", "Size", "Region", "Filesystem Type", "Filesystem Label", "Droplet IDs",
	}
}

func (a *volume) ColMap() map[string]string {
	return map[string]string{
		"ID":               "ID",
		"Name":             "Name",
		"Size":             "Size",
		"Region":           "Region",
		"Filesystem Type":  "Filesystem Type",
		"Filesystem Label": "Filesystem Label",
		"Droplet IDs":      "Droplet IDs",
	}

}

func (a *volume) KV() []map[string]interface{} {
	out := []map[string]interface{}{}
	for _, volume := range a.volumes {

		m := map[string]interface{}{
			"ID":               volume.ID,
			"Name":             volume.Name,
			"Size":             strconv.FormatInt(volume.SizeGigaBytes, 10) + " GiB",
			"Region":           volume.Region.Slug,
			"Filesystem Type":  volume.FilesystemType,
			"Filesystem Label": volume.FilesystemLabel,
		}
		m["DropletIDs"] = ""
		if len(volume.DropletIDs) != 0 {
			m["DropletIDs"] = fmt.Sprintf("%v", volume.DropletIDs)
		}
		out = append(out, m)

	}
	return out

}

type snapshot struct {
	snapshots do.Snapshots
}

var _ Displayable = &snapshot{}

func (s *snapshot) JSON(out io.Writer) error {
	return writeJSON(s.snapshots, out)
}

func (s *snapshot) Cols() []string {
	return []string{"ID", "Name", "CreatedAt", "Regions", "ResourceId",
		"ResourceType", "MinDiskSize", "Size"}
}

func (s *snapshot) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "CreatedAt": "Created at", "Regions": "Regions",
		"ResourceId": "Resource ID", "ResourceType": "Resource Type", "MinDiskSize": "Min Disk Size", "Size": "Size"}
}

func (s *snapshot) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, ss := range s.snapshots {
		o := map[string]interface{}{
			"ID": ss.ID, "Name": ss.Name, "ResourceId": ss.ResourceID,
			"ResourceType": ss.ResourceType, "Regions": ss.Regions, "MinDiskSize": ss.MinDiskSize,
			"Size": strconv.FormatFloat(ss.SizeGigaBytes, 'f', 2, 64) + " GiB", "CreatedAt": ss.Created,
		}
		out = append(out, o)
	}

	return out
}

type certificate struct {
	certificates do.Certificates
}

var _ Displayable = &certificate{}

func (c *certificate) JSON(out io.Writer) error {
	return writeJSON(c.certificates, out)
}

func (c *certificate) Cols() []string {
	return []string{
		"ID",
		"Name",
		"DNSNames",
		"SHA1Fingerprint",
		"NotAfter",
		"Created",
		"Type",
		"State",
	}
}

func (c *certificate) ColMap() map[string]string {
	return map[string]string{
		"ID":              "ID",
		"Name":            "Name",
		"DNSNames":        "DNS Names",
		"SHA1Fingerprint": "SHA-1 Fingerprint",
		"NotAfter":        "Expiration Date",
		"Created":         "Created At",
		"Type":            "Type",
		"State":           "State",
	}
}

func (c *certificate) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, c := range c.certificates {
		o := map[string]interface{}{
			"ID":              c.ID,
			"Name":            c.Name,
			"DNSNames":        fmt.Sprintf(strings.Join(c.DNSNames, ",")),
			"SHA1Fingerprint": c.SHA1Fingerprint,
			"NotAfter":        c.NotAfter,
			"Created":         c.Created,
			"Type":            c.Type,
			"State":           c.State,
		}
		out = append(out, o)
	}

	return out
}

type loadBalancer struct {
	loadBalancers do.LoadBalancers
}

var _ Displayable = &loadBalancer{}

func (lb *loadBalancer) JSON(out io.Writer) error {
	return writeJSON(lb.loadBalancers, out)
}

func (lb *loadBalancer) Cols() []string {
	return []string{
		"ID",
		"IP",
		"Name",
		"Status",
		"Created",
		"Algorithm",
		"Region",
		"Tag",
		"DropletIDs",
		"RedirectHttpToHttps",
		"StickySessions",
		"HealthCheck",
		"ForwardingRules",
	}
}

func (lb *loadBalancer) ColMap() map[string]string {
	return map[string]string{
		"ID":                  "ID",
		"IP":                  "IP",
		"Name":                "Name",
		"Status":              "Status",
		"Created":             "Created At",
		"Algorithm":           "Algorithm",
		"Region":              "Region",
		"Tag":                 "Tag",
		"DropletIDs":          "Droplet IDs",
		"RedirectHttpToHttps": "SSL",
		"StickySessions":      "Sticky Sessions",
		"HealthCheck":         "Health Check",
		"ForwardingRules":     "Forwarding Rules",
	}
}

func (lb *loadBalancer) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, l := range lb.loadBalancers {
		forwardingRules := []string{}
		for _, r := range l.ForwardingRules {
			forwardingRules = append(forwardingRules, prettyPrintStruct(r))
		}

		o := map[string]interface{}{
			"ID":                  l.ID,
			"IP":                  l.IP,
			"Name":                l.Name,
			"Status":              l.Status,
			"Created":             l.Created,
			"Algorithm":           l.Algorithm,
			"Region":              l.Region.Slug,
			"Tag":                 l.Tag,
			"DropletIDs":          fmt.Sprintf(strings.Trim(strings.Replace(fmt.Sprint(l.DropletIDs), " ", ",", -1), "[]")),
			"RedirectHttpToHttps": l.RedirectHttpToHttps,
			"StickySessions":      prettyPrintStruct(l.StickySessions),
			"HealthCheck":         prettyPrintStruct(l.HealthCheck),
			"ForwardingRules":     fmt.Sprintf(strings.Join(forwardingRules, " ")),
		}
		out = append(out, o)
	}

	return out
}

type firewall struct {
	firewalls do.Firewalls
}

var _ Displayable = &firewall{}

func (f *firewall) JSON(out io.Writer) error {
	return writeJSON(f.firewalls, out)
}

func (f *firewall) Cols() []string {
	return []string{
		"ID",
		"Name",
		"Status",
		"Created",
		"InboundRules",
		"OutboundRules",
		"DropletIDs",
		"Tags",
		"PendingChanges",
	}
}

func (f *firewall) ColMap() map[string]string {
	return map[string]string{
		"ID":             "ID",
		"Name":           "Name",
		"Status":         "Status",
		"Created":        "Created At",
		"InboundRules":   "Inbound Rules",
		"OutboundRules":  "Outbound Rules",
		"DropletIDs":     "Droplet IDs",
		"Tags":           "Tags",
		"PendingChanges": "Pending Changes",
	}
}

func (f *firewall) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, fw := range f.firewalls {
		irs, ors := firewallRulesPrintHelper(fw)
		o := map[string]interface{}{
			"ID":             fw.ID,
			"Name":           fw.Name,
			"Status":         fw.Status,
			"Created":        fw.Created,
			"InboundRules":   irs,
			"OutboundRules":  ors,
			"DropletIDs":     dropletListHelper(fw.DropletIDs),
			"Tags":           strings.Join(fw.Tags, ","),
			"PendingChanges": firewallPendingChangesPrintHelper(fw),
		}
		out = append(out, o)
	}

	return out
}

func firewallRulesPrintHelper(fw do.Firewall) (string, string) {
	var irs, ors []string

	for _, ir := range fw.InboundRules {
		ss := firewallInAndOutboundRulesPrintHelper(ir.Sources.Addresses, ir.Sources.Tags, ir.Sources.DropletIDs, ir.Sources.LoadBalancerUIDs)
		if ir.Protocol == "icmp" {
			irs = append(irs, fmt.Sprintf("%v:%v,%v", "protocol", ir.Protocol, ss))
		} else {
			irs = append(irs, fmt.Sprintf("%v:%v,%v:%v,%v", "protocol", ir.Protocol, "ports", ir.PortRange, ss))
		}
	}

	for _, or := range fw.OutboundRules {
		ds := firewallInAndOutboundRulesPrintHelper(or.Destinations.Addresses, or.Destinations.Tags, or.Destinations.DropletIDs, or.Destinations.LoadBalancerUIDs)
		if or.Protocol == "icmp" {
			ors = append(ors, fmt.Sprintf("%v:%v,%v", "protocol", or.Protocol, ds))
		} else {
			ors = append(ors, fmt.Sprintf("%v:%v,%v:%v,%v", "protocol", or.Protocol, "ports", or.PortRange, ds))
		}
	}

	return strings.Join(irs, " "), strings.Join(ors, " ")
}

func firewallInAndOutboundRulesPrintHelper(addresses []string, tags []string, dropletIDs []int, loadBalancerUIDs []string) string {
	output := []string{}
	resources := map[string][]string{
		"address":           addresses,
		"tag":               tags,
		"load_balancer_uid": loadBalancerUIDs,
	}

	for k, vs := range resources {
		for _, r := range vs {
			output = append(output, fmt.Sprintf("%v:%v", k, r))
		}
	}

	for _, dID := range dropletIDs {
		output = append(output, fmt.Sprintf("%v:%v", "droplet_id", dID))
	}

	return strings.Join(output, ",")
}

func firewallPendingChangesPrintHelper(fw do.Firewall) string {
	output := []string{}

	for _, pc := range fw.PendingChanges {
		output = append(output, fmt.Sprintf("%v:%v,%v:%v,%v:%v", "droplet_id", pc.DropletID, "removing", pc.Removing, "status", pc.Status))
	}

	return strings.Join(output, " ")
}

func dropletListHelper(IDs []int) string {
	output := []string{}

	for _, id := range IDs {
		output = append(output, strconv.Itoa(id))
	}

	return strings.Join(output, ",")
}

func prettyPrintStruct(obj interface{}) string {
	output := []string{}

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovered from %v", err)
		}
	}()

	val := reflect.Indirect(reflect.ValueOf(obj))
	for i := 0; i < val.NumField(); i++ {
		k := strings.Split(val.Type().Field(i).Tag.Get("json"), ",")[0]
		v := reflect.ValueOf(val.Field(i).Interface())
		output = append(output, fmt.Sprintf("%v:%v", k, v))
	}

	return fmt.Sprintf(strings.Join(output, ","))
}
