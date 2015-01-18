package apiv2

import (
	"errors"
)

const (
	DefaultSizeSlug = "512mb"
)

// slug				string	A human-readable string that is used to uniquely identify each size.
// memory			number	The amount of RAM available to Droplets created with this size. This value is given in megabytes.
// vcpus			number	The number of virtual CPUs that are allocated to Droplets with this size.
// disk				number	This is the amount of disk space set aside for Droplets created with this size. The value is given in gigabytes.
// transfer			number	The amount of transfer bandwidth that is available for Droplets created in this size. This only counts traffic on the public interface. The value is given in terabytes.
// price_monthly	number	This attribute describes the monthly cost of this Droplet size if the Droplet is kept for an entire month. The value is measured in US dollars.
// price_hourly		number	This describes the price of the Droplet size as measured hourly. The value is measured in US dollars.
// regions			array	An array that contains the region slugs where this size is available for Droplet creates.
type Size struct {
	Slug         string   `json:"slug"`
	Memory       int      `json:"memory"`
	VCPUS        int      `json:"vcpus"`
	Disk         int      `json:"disk"`
	Transfer     float32  `json:"transfer"`
	PriceMonthly float32  `json:"price_monthly"`
	PriceHourly  float32  `json:"price_hourly"`
	Regions      []string `json:"regions"`
}

type SizeListResponse struct {
	Sizes []*Size `json:"sizes"`
	Meta  struct {
		Total int `json:"total"`
	} `json:"meta"`
}

func NewSize() *Size {
	return &Size{
		Slug: DefaultSizeSlug,
	}
}

func (c *Client) LoadSize(name string) (*Size, error) {
	var sizeList SizeListResponse

	err := c.Get("sizes", nil, &sizeList, nil)
	if err != nil {
		return nil, errors.New(err.Message)
	}

	for _, size := range sizeList.Sizes {
		if size.Slug == name {
			return size, nil
		}
	}

	return nil, errors.New("Size not found.")
}

func (c *Client) ListAllSizes() (*SizeListResponse, error) {
	var sizeList *SizeListResponse

	err := c.Get("sizes", nil, &sizeList, nil)
	if err != nil {
		return nil, errors.New(err.Message)
	}

	return sizeList, nil
}
