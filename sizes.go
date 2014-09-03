package godo

// SizesService handles communication with the size related methods of the
// DigitalOcean API.
type SizesService struct {
	client *Client
}

// Size represents a DigitalOcean Size
type Size struct {
	Slug         string   `json:"slug,omitempty"`
	Memory       int      `json:"memory,omitempty"`
	Vcpus        int      `json:"vcpus,omitempty"`
	Disk         int      `json:"disk,omitempty"`
	PriceMonthly float64  `json:"price_monthly,omitempty"`
	PriceHourly  float64  `json:"price_hourly,omitempty"`
	Regions      []string `json:"regions,omitempty"`
}

func (s Size) String() string {
	return Stringify(s)
}

type sizesRoot struct {
	Sizes []Size
}

// List all images
func (s *SizesService) List() ([]Size, *Response, error) {
	path := "v2/sizes"

	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	sizes := new(sizesRoot)
	resp, err := s.client.Do(req, sizes)
	if err != nil {
		return nil, resp, err
	}

	return sizes.Sizes, resp, err
}
