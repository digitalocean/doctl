package apiv2

import (
	"errors"
)

const (
	DefaultSizeSlug = "512mb"
)

type Size struct {
	Slug         string   `json:"slug"`
	Transfer     int      `json:"transfer"`
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
