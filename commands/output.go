package commands

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/digitalocean/godo"
)

func newTabWriter(out io.Writer) *tabwriter.Writer {
	w := new(tabwriter.Writer)
	w.Init(out, 0, 8, 1, '\t', 0)

	return w
}

type account struct {
	*godo.Account
}

var _ displayer = &account{}

func (a *account) JSON(out io.Writer) error {
	return writeJSON(a.Account, out)
}

func (a *account) String(out io.Writer) error {
	account := a.Account

	w := newTabWriter(out)

	fmt.Fprintln(w, "Email\tDroplet Limit\tEmail Verified\tUUID\tStatus")
	fmt.Fprintf(w, "")
	fmt.Fprintf(w, "%s\t%d\t%t\t%s\t%s\n", account.Email, account.DropletLimit, account.EmailVerified, account.UUID, account.Status)
	fmt.Fprintln(w)
	return w.Flush()
}

type actions []godo.Action

type action struct {
	actions
}

var _ displayer = &action{}

func (a *action) JSON(out io.Writer) error {
	return writeJSON(a.actions, out)
}

func (a *action) String(out io.Writer) error {
	w := newTabWriter(out)

	fmt.Fprintln(w, "ID\tStatus\tType\tStarted At\tCompleted At\tResource ID\tResource Type\tRegion")

	for _, a := range a.actions {
		fmt.Fprintf(w, "")
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			a.ID, a.Status, a.Type, a.StartedAt, a.CompletedAt, a.ResourceID, a.ResourceType, a.RegionSlug)
	}
	fmt.Fprintln(w)
	return w.Flush()
}

type domains []godo.Domain

type domain struct {
	domains
}

var _ displayer = &domain{}

func (d *domain) JSON(out io.Writer) error {
	return writeJSON(d.domains, out)
}

func (d *domain) String(out io.Writer) error {
	w := newTabWriter(out)

	if len(d.domains) == 1 {
		fmt.Fprintln(out, d.domains[0].ZoneFile)
		return nil
	}

	fmt.Fprintln(w, "Name")

	for _, d := range d.domains {
		fmt.Fprintf(w, "%s\n", d.Name)
	}

	fmt.Fprintln(w)
	return w.Flush()
}

type domainRecords []godo.DomainRecord

type domainRecord struct {
	domainRecords
}

func (dr *domainRecord) JSON(out io.Writer) error {
	return writeJSON(dr.domainRecords, out)
}

func (dr *domainRecord) String(out io.Writer) error {
	w := newTabWriter(out)

	fmt.Fprintln(w, "ID\tType\tName\tData\tPriority\tPort\tWeight")

	for _, d := range dr.domainRecords {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\t%d\n", d.ID, d.Type, d.Name, d.Data,
			d.Priority, d.Port, d.Weight)
	}

	fmt.Fprintln(w)
	return w.Flush()
}

type droplets []godo.Droplet

type droplet struct {
	droplets
}

var _ displayer = &droplet{}

func (d *droplet) JSON(out io.Writer) error {
	return writeJSON(d.droplets, out)
}

func (d *droplet) String(out io.Writer) error {
	w := newTabWriter(out)

	fmt.Fprintln(w, "ID\tName\tPublic IPv4\tMemory\tVCPUs\tDisk\tRegion\tImage\tStatus")

	for _, d := range d.droplets {
		ips := extractDropletIPs(&d)
		image := fmt.Sprintf("%s %s", d.Image.Distribution, d.Image.Name)
		fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\t%s\t%s\t%s\n",
			d.ID, d.Name, ips[ifacePublic], d.Memory, d.Vcpus, d.Disk, d.Region.Slug, image, d.Status)
	}
	fmt.Fprintln(w)
	return w.Flush()
}

type floatingIPs []godo.FloatingIP

type floatingIP struct {
	floatingIPs
}

var _ displayer = &floatingIP{}

func (fi *floatingIP) JSON(out io.Writer) error {
	return writeJSON(fi.floatingIPs, out)
}

func (fi *floatingIP) String(out io.Writer) error {
	w := newTabWriter(out)

	fmt.Fprintln(w, "IP\tRegion\tDroplet ID\tDroplet Name")
	for _, ip := range fi.floatingIPs {
		var dropletID, dropletName string
		if ip.Droplet != nil {
			dropletID = fmt.Sprintf("%d", ip.Droplet.ID)
			dropletName = ip.Droplet.Name
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ip.IP, ip.Region.Slug, dropletID, dropletName)
	}

	fmt.Fprintln(w)
	return w.Flush()
}

type images []godo.Image

type image struct {
	images
}

var _ displayer = &image{}

func (gi *image) JSON(out io.Writer) error {
	return writeJSON(gi.images, out)
}

func (gi *image) String(out io.Writer) error {
	w := newTabWriter(out)

	fmt.Fprintln(w, "ID\tName\tType\tDistribution\tSlug\tPublic\tMin Disk")

	for _, i := range gi.images {
		publicStatus := false
		if i.Public {
			publicStatus = true
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%t\t%d\n",
			i.ID, i.Name, i.Type, i.Distribution, i.Slug, publicStatus, i.MinDiskSize)

	}
	fmt.Fprintln(w)
	return w.Flush()
}

type kernels []godo.Kernel

type kernel struct {
	kernels
}

var _ displayer = &kernel{}

func (ke *kernel) JSON(out io.Writer) error {
	return writeJSON(ke.kernels, out)
}

func (ke *kernel) String(out io.Writer) error {
	w := newTabWriter(out)

	fmt.Fprintln(w, "ID\tName\tVersion")

	for _, k := range ke.kernels {
		fmt.Fprintf(w, "%d\t%s\t%s\n", k.ID, k.Name, k.Version)
	}
	fmt.Fprintln(w)
	return w.Flush()
}

type keys []godo.Key

type key struct {
	keys
}

var _ displayer = &key{}

func (ke *key) JSON(out io.Writer) error {
	return writeJSON(ke.keys, out)
}

func (ke *key) String(out io.Writer) error {
	w := newTabWriter(out)

	fmt.Fprintln(w, "ID\tName\tFingerprint")

	for _, s := range ke.keys {
		fmt.Fprintf(w, "%d\t%s\t%s\n",
			s.ID, s.Name, s.Fingerprint)
	}
	fmt.Fprintln(w)
	return w.Flush()
}

type regions []godo.Region

type region struct {
	regions
}

var _ displayer = &region{}

func (re *region) JSON(out io.Writer) error {
	return writeJSON(re.regions, out)
}

func (re *region) String(out io.Writer) error {
	w := newTabWriter(out)

	fmt.Fprintln(w, "Slug\tName\tAvailable")

	for _, r := range re.regions {
		fmt.Fprintf(w, "%s\t%s\t%t\n", r.Slug, r.Name, r.Available)
	}
	fmt.Fprintln(w)
	return w.Flush()
}

type sizes []godo.Size

type size struct {
	sizes
}

var _ displayer = &size{}

func (si *size) JSON(out io.Writer) error {
	return writeJSON(si.sizes, out)
}

func (si *size) String(out io.Writer) error {
	w := newTabWriter(out)

	fmt.Fprintln(w, "Slug\tMemory\tVcpus\tDisk\tPrice Monthly\tPrice Hourly")

	for _, s := range si.sizes {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%0.2f\t%f\n",
			s.Slug, s.Memory, s.Vcpus, s.Disk, s.PriceMonthly, s.PriceHourly)
	}
	fmt.Fprintln(w)
	return w.Flush()
}
