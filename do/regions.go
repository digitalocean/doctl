package do

import "github.com/digitalocean/godo"

// Region wraps godo Region.
type Region struct {
	*godo.Region
}

// Regions is a slice of Region.
type Regions []Region

// RegionsService is the godo RegionsService interface.
type RegionsService interface {
	List() (Regions, error)
}

type regionsService struct {
	client *godo.Client
}

var _ RegionsService = &regionsService{}

// NewRegionsService builds an instance of RegionsService.
func NewRegionsService(client *godo.Client) RegionsService {
	return &regionsService{
		client: client,
	}
}

func (rs *regionsService) List() (Regions, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := rs.client.Regions.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Regions, len(si))
	for i := range si {
		r := si[i].(godo.Region)
		list[i] = Region{Region: &r}
	}

	return list, nil
}
