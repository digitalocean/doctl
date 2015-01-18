package apiv2

import (
	"fmt"
)

const (
	DefaultImageSlug = "5914637"
)

// id				number	A unique identifier for each Droplet instance. This is automatically generated upon Droplet creation.
// name				string	The human-readable name set for the Droplet instance.
// memory			number	Memory of the Droplet in megabytes.
// vcpus			number	The number of virtual CPUs.
// disk				number	The size of the Droplet's disk in gigabytes.
// region			object	The region that the Droplet instance is deployed in. When setting a region, the value should be the slug identifier for the region. When you query a Droplet, the entire region object will be returned.
// image			object	The base image used to create the Droplet instance. When setting an image, the value is set to the image id or slug. When querying the Droplet, the entire image object will be returned.
// kernel			object	The current kernel. This will initially be set to the kernel of the base image when the Droplet is created.
// size				object	The size of the Droplet instance. When setting a size, the value should be the slug identifier for a particular size. When querying the Droplet, a partial size object will be returned.
// locked			boolean	A boolean value indicating whether the Droplet has been locked, preventing actions by users.
// created_at		string	A time value given in ISO8601 combined date and time format that represents when the Droplet was created.
// status			string	A status string indicating the state of the Droplet instance. This may be "new", "active", "off", or "archive".
// networks			object	The details of the network that are configured for the Droplet instance. This is an object that contains keys for IPv4 and IPv6. The value of each of these is an array that contains objects describing an individual IP resource allocated to the Droplet. These will define attributes like the IP address, netmask, and gateway of the specific network depending on the type of network it is.
// backup_ids		array	An array of backup IDs of any backups that have been taken of the Droplet instance. Droplet backups are enabled at the time of the instance creation.
// snapshot_ids		array	An array of snapshot IDs of any snapshots created from the Droplet instance.
// features			array	An array of features enabled on this Droplet.
type Droplet struct {
	ID          int                    `json:"id,omitempty"`
	Name        string                 `json:"name"`
	Memory      int                    `json:"memory"`
	VCPUs       int                    `json:"vcpus"`
	Disk        int                    `json:"disk"`
	Region      *Region                `json:"region"`
	Image       *Image                 `json:"image"`
	Kernel      *Kernel                `json:"kernel"`
	Size        *Size                  `json:"size"`
	Locked      bool                   `json:"locked"`
	CreatedAt   string                 `json:"created_at"`
	Status      string                 `json:"status"`
	Networks    map[string]NetworkList `json:"networks"`
	BackupIDs   []int                  `json:"backup_ids"`
	SnapshotIDs []int                  `json:"snapshot_ids"`
	Features    []string               `json:"features"`
	Client      *Client
}

// name					String	The human-readable string you wish to use when displaying the Droplet name. The name, if set to a domain name managed in the DigitalOcean DNS management system, will configure a PTR record for the Droplet. The name set during creation will also determine the hostname for the Droplet in its internal configuration.	Yes
// region				String	The unique slug identifier for the region that you wish to deploy in.	Yes
// size					String	The unique slug identifier for the size that you wish to select for this Droplet.	Yes
// image				number (if using an image ID), or String (if using a public image slug)	The image ID of a public or private image, or the unique slug identifier for a public image. This image will be the base image for your Droplet.	Yes
// ssh_keys				Array	An array containing the IDs or fingerprints of the SSH keys that you wish to embed in the Droplet's root account upon creation.	No
// backups				Boolean	A boolean indicating whether automated backups should be enabled for the Droplet. Automated backups can only be enabled when the Droplet is created.	No
// ipv6					Boolean	A boolean indicating whether IPv6 is enabled on the Droplet.	No
// private_networking	Boolean	A boolean indicating whether private networking is enabled for the Droplet. Private networking is currently only available in certain regions.	No
// user_data			String	A string of the desired User Data for the Droplet. User Data is currently only available in regions with metadata listed in their features.
type DropletRequest struct {
	Name              string   `json:"name"`
	Region            string   `json:"region"`
	Size              string   `json:"size"`
	Image             string   `json:"image"`
	SSHKeys           []string `json:"ssh_keys"`
	Backups           bool     `json:"backups"`
	IPv6              bool     `json:"ipv6"`
	PrivateNetworking bool     `json:"private_networking"`
	UserData          string   `json:"user_data"`
}

type Image struct {
	ID           int      `json:"id,omitempty"`
	Name         string   `json:"name"`
	Distribution string   `json:"distribution"`
	Slug         string   `json:"slug"`
	Public       bool     `json:"public"`
	Regions      []string `json:"regions"`
}

type Kernel struct {
	ID      int    `json:"id,omitempty"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Network struct {
	IPAddress string `json:"ip_address"`
	Netmask   string `json:"netmask"`
	Gateway   string `json:"gateway"`
	Type      string `json:"type"`
}

type NetworkList []*Network

type DropletList struct {
	Droplets []*Droplet `json:"droplets"`
	Meta     struct {
		Total int `json:"total"`
	} `json:"meta"`
}

type DropletResponse struct {
	Droplet *Droplet `json:"droplet"`
	Links   struct {
		Actions []struct {
			ID   int    `json:"id"`
			Rel  string `json:"rel"`
			HREF string `json:"href"`
		} `json:"actions"`
	} `json:"links"`
}

func (c *Client) NewDroplet() *Droplet {
	return &Droplet{
		Client: c,
	}
}

func (c *Client) NewDropletRequest(name string) *DropletRequest {
	dr := &DropletRequest{
		Name:              name,
		Region:            DefaultRegionSlug,
		Size:              DefaultSizeSlug,
		Image:             DefaultImageSlug,
		Backups:           false,
		IPv6:              false,
		PrivateNetworking: false,
		UserData:          "",
	}

	return dr
}

func (c *Client) CreateDroplet(request *DropletRequest) (*Droplet, error) {
	var dropletResponse DropletResponse

	apiErr := c.Post("droplets", request, &dropletResponse, nil)
	if apiErr != nil {
		return nil, fmt.Errorf("API Error: %s", apiErr.Message)
	}

	return dropletResponse.Droplet, nil
}

func (c *Client) ListDroplets() (*DropletList, error) {
	var dropletList DropletList

	apiErr := c.Get("droplets", nil, &dropletList, nil)
	if apiErr != nil {
		return nil, fmt.Errorf("API Error: %s", apiErr.Message)
	}

	return &dropletList, nil
}

func (c *Client) DestroyDropletByID(id int) error {
	err := c.Delete(fmt.Sprintf("droplets/%d", id), nil, nil)
	if err != nil {
		return fmt.Errorf("API Error: %s", err.Message)
	}
	return nil
}

func (c *Client) DestroyDropletByName(name string) error {
	dropletList, err := c.ListDroplets()
	if err != nil {
		return err
	}

	for _, droplet := range dropletList.Droplets {
		if droplet.Name == name {
			apiErr := c.Delete(fmt.Sprintf("droplets/%d", droplet.ID), nil, nil)
			if apiErr != nil {
				return fmt.Errorf("API Error: %s", apiErr.Message)
			}
			return nil
		}
	}

	return fmt.Errorf("%s Not Found.", name)
}

func (c *Client) FindDropletByName(name string) (*Droplet, error) {
	dropletList, err := c.ListDroplets()
	if err != nil {
		return nil, err
	}

	for _, droplet := range dropletList.Droplets {
		if droplet.Name == name {
			droplet.Client = c
			return droplet, nil
		}
	}

	return nil, fmt.Errorf("%s Not Found.", name)
}

func (c *Client) FindDropletByID(id int) (*Droplet, error) {
	return nil, fmt.Errorf("API Error: Unable to find Droplet %s.", id)
}

func (d *Droplet) PublicIPAddress() string {
	var ipProtocol string
	var ipIdx int
	var ipAddress string
	for net, networks := range d.Networks {
		for idx, network := range networks {
			if network.Type == "public" {
				ipProtocol = net
				ipIdx = idx
			}
		}
	}

	if len(d.Networks[ipProtocol]) > 0 {
		ipAddress = d.Networks[ipProtocol][ipIdx].IPAddress
	} else {
		ipAddress = "0.0.0.0"
	}

	return ipAddress
}
