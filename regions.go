package godo

// RegionsService handles communication with the region related methods of the
// DigitalOcean API.
type RegionsService struct {
	client *Client
}

// Region represents a DigitalOcean Region
type Region struct {
	Slug      string   `json:"slug,omitempty"`
	Name      string   `json:"name,omitempty"`
	Sizes     []string `json:"sizes,omitempty"`
	Available bool     `json:"available,omitempty`
}

type regionsRoot struct {
	Regions []Region
}

type regionRoot struct {
	Region Region
}

func (r Region) String() string {
	return Stringify(r)
}

// List all regions
func (s *RegionsService) List() ([]Region, *Response, error) {
	path := "v2/regions"

	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	regions := new(regionsRoot)
	resp, err := s.client.Do(req, regions)
	if err != nil {
		return nil, resp, err
	}

	return regions.Regions, resp, err
}
