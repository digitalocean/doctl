package commands

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/digitalocean/godo"
)

var (
	hc = &headerControl{}
)

func newTabWriter(out io.Writer) *tabwriter.Writer {
	w := new(tabwriter.Writer)
	w.Init(out, 0, 8, 1, '\t', 0)

	return w
}

type headerControl struct {
	hideHeader bool
}

func (hc *headerControl) HideHeader(hide bool) {
	hc.hideHeader = hide
}

type account struct {
	*godo.Account
}

var _ displayer = &account{}

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

type actions []godo.Action

type action struct {
	actions
}

var _ displayer = &action{}

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
		o := map[string]interface{}{
			"ID": x.ID, "Status": x.Status, "Type": x.Type,
			"StartedAt": x.StartedAt, "CompletedAt": x.CompletedAt,
			"ResourceID": x.ResourceID, "ResourceType": x.ResourceType,
			"Region": x.Region.Slug,
		}
		out = append(out, o)
	}

	return out
}

type domains []godo.Domain

type domain struct {
	domains
}

var _ displayer = &domain{}

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

type domainRecords []godo.DomainRecord

type domainRecord struct {
	domainRecords
}

func (dr *domainRecord) JSON(out io.Writer) error {
	return writeJSON(dr.domainRecords, out)
}

func (dr *domainRecord) Cols() []string {
	return []string{
		"ID", "Type", "Name", "Data", "Priority", "Port", "Weight",
	}
}

func (dr *domainRecord) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Type": "Type", "Name": "Name", "Data": "Data",
		"Priority": "Priority", "Port": "Port", "Weight": "Weight",
	}
}

func (dr *domainRecord) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, d := range dr.domainRecords {
		o := map[string]interface{}{
			"ID": d.ID, "Type": d.Type, "Name": d.Name,
			"Data": d.Data, "Priority": d.Priority,
			"Port": d.Port, "Weight": d.Weight,
		}
		out = append(out, o)
	}

	return out
}

type droplets []godo.Droplet

type droplet struct {
	droplets
}

var _ displayer = &droplet{}

func (d *droplet) JSON(out io.Writer) error {
	return writeJSON(d.droplets, out)
}

func (d *droplet) Cols() []string {
	return []string{
		"ID", "Name", "PublicIPv4", "Memory", "VCPUs", "Disk", "Region", "Image", "Status",
	}
}

func (d *droplet) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "PublicIPv4": "Public IPv4",
		"Memory": "Memory", "VCPUs": "VCPUs", "Disk": "Disk",
		"Region": "Region", "Image": "Image", "Status": "Status",
	}
}

func (d *droplet) KV() []map[string]interface{} {
	out := []map[string]interface{}{}
	for _, d := range d.droplets {
		ips := extractDropletIPs(&d)
		image := fmt.Sprintf("%s %s", d.Image.Distribution, d.Image.Name)
		m := map[string]interface{}{
			"ID": d.ID, "Name": d.Name, "PublicIPv4": ips[ifacePublic],
			"Memory": d.Memory, "VCPUs": d.Vcpus, "Disk": d.Disk,
			"Region": d.Region.Slug, "Image": image, "Status": d.Status,
		}
		out = append(out, m)
	}

	return out
}

type floatingIPs []godo.FloatingIP

type floatingIP struct {
	floatingIPs
}

var _ displayer = &floatingIP{}

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

type images []godo.Image

type image struct {
	images
}

var _ displayer = &image{}

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

type kernels []godo.Kernel

type kernel struct {
	kernels
}

var _ displayer = &kernel{}

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

type keys []godo.Key

type key struct {
	keys
}

var _ displayer = &key{}

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

type regions []godo.Region

type region struct {
	regions
}

var _ displayer = &region{}

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

type sizes []godo.Size

type size struct {
	sizes
}

var _ displayer = &size{}

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
