package apiv2

import (
	"errors"
)

const (
	RegionNYC1 = "nyc1"
	RegionNYC2 = "nyc2"
	RegionNYC3 = "nyc3"
	RegionSFO1 = "sfo1"
	RegionAMS1 = "ams1"
	RegionAMS2 = "ams2"
	RegionSGP1 = "sgp1"
	RegionLON1 = "lon1"

	DefaultRegionSlug = RegionNYC3
)

type Region struct {
	Slug      string   `json:"slug"`
	Name      string   `json:"name"`
	Sizes     []string `json:"sizes"`
	Available bool     `json:"available"`
	Features  []string `json:"features"`
	client    *Client
}

func NewRegion() *Region {
	return &Region{
		Slug: DefaultRegionSlug,
	}
}

func (c *Client) LoadRegion(name string) (*Region, error) {
	var regionList []*Region

	err := c.Get("regions", nil, &regionList, nil)
	if err != nil {
		return nil, errors.New(err.Message)
	}

	for _, region := range regionList {
		if region.Slug == name {
			return region, nil
		}
	}

	return nil, errors.New("Region not found.")
}
