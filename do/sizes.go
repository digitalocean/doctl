package do

import (
	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
)

// Size wraps godo Size.
type Size struct {
	*godo.Size
}

// Sizes is a slice of Size.
type Sizes []Size

// SizesService is the godo SizesService interface.
type SizesService interface {
	List() (Sizes, error)
}

type sizesService struct {
	client *godo.Client
}

var _ SizesService = &sizesService{}

// NewSizesService builds an instance of SizesService.
func NewSizesService(client *godo.Client) SizesService {
	return &sizesService{
		client: client,
	}
}

func (rs *sizesService) List() (Sizes, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := rs.client.Sizes.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Sizes, len(si))
	for i := range si {
		r := si[i].(godo.Size)
		list[i] = Size{Size: &r}
	}

	return list, nil
}
