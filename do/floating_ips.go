package do

import "github.com/digitalocean/godo"

// FloatingIP wraps a godo FloatingIP.
type FloatingIP struct {
	*godo.FloatingIP
}

// FloatingIPs is a slice of FloatingIP.
type FloatingIPs []FloatingIP

// FloatingIPsService is the godo FloatingIPsService interface.
type FloatingIPsService interface {
	List() (FloatingIPs, error)
	Get(ip string) (*FloatingIP, error)
	Create(ficr *godo.FloatingIPCreateRequest) (*FloatingIP, error)
	Delete(ip string) error
}

type floatingIPsService struct {
	client *godo.Client
}

var _ FloatingIPsService = &floatingIPsService{}

// NewFloatingIPsService builds an instance of FloatingIPsService.
func NewFloatingIPsService(client *godo.Client) FloatingIPsService {
	return &floatingIPsService{
		client: client,
	}
}

func (fis *floatingIPsService) List() (FloatingIPs, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := fis.client.FloatingIPs.List(opt)
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

	var list FloatingIPs
	for _, x := range si {
		fip := x.(godo.FloatingIP)
		list = append(list, FloatingIP{FloatingIP: &fip})
	}

	return list, nil
}

func (fis *floatingIPsService) Get(ip string) (*FloatingIP, error) {
	fip, _, err := fis.client.FloatingIPs.Get(ip)
	if err != nil {
		return nil, err
	}

	return &FloatingIP{FloatingIP: fip}, nil
}

func (fis *floatingIPsService) Create(ficr *godo.FloatingIPCreateRequest) (*FloatingIP, error) {
	fip, _, err := fis.client.FloatingIPs.Create(ficr)
	if err != nil {
		return nil, err
	}

	return &FloatingIP{FloatingIP: fip}, nil
}

func (fis *floatingIPsService) Delete(ip string) error {
	_, err := fis.client.FloatingIPs.Delete(ip)
	return err
}
