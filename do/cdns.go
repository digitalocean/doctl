package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// CDN is a wrapper for godo.CDN.
type CDN struct {
	*godo.CDN
}

// CDNsService is an interface for interacting with DigitalOcean's CDN api.
type CDNsService interface {
	List() ([]CDN, error)
	Get(string) (*CDN, error)
	Create(*godo.CDNCreateRequest) (*CDN, error)
	UpdateTTL(string, *godo.CDNUpdateTTLRequest) (*CDN, error)
	UpdateCustomDomain(string, *godo.CDNUpdateCustomDomainRequest) (*CDN, error)
	FlushCache(string, *godo.CDNFlushCacheRequest) error
	Delete(string) error
}

type cdnsService struct {
	client *godo.Client
}

// NewCDNsService builds an NewCDNsService instance.
func NewCDNsService(godoClient *godo.Client) CDNsService {
	return &cdnsService{
		client: godoClient,
	}
}

func (c *cdnsService) List() ([]CDN, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := c.client.CDNs.List(context.TODO(), opt)
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

	list := make([]CDN, len(si))
	for i := range si {
		c := si[i].(godo.CDN)
		list[i] = CDN{CDN: &c}
	}
	return list, nil
}

func (c *cdnsService) Get(id string) (*CDN, error) {
	cdn, _, err := c.client.CDNs.Get(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	return &CDN{CDN: cdn}, nil
}

func (c *cdnsService) Create(req *godo.CDNCreateRequest) (*CDN, error) {
	cdn, _, err := c.client.CDNs.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	return &CDN{CDN: cdn}, nil
}

func (c *cdnsService) UpdateTTL(id string, req *godo.CDNUpdateTTLRequest) (*CDN, error) {
	cdn, _, err := c.client.CDNs.UpdateTTL(context.TODO(), id, req)
	if err != nil {
		return nil, err
	}

	return &CDN{CDN: cdn}, nil
}

func (c *cdnsService) UpdateCustomDomain(id string, req *godo.CDNUpdateCustomDomainRequest) (*CDN, error) {
	cdn, _, err := c.client.CDNs.UpdateCustomDomain(context.TODO(), id, req)
	if err != nil {
		return nil, err
	}

	return &CDN{CDN: cdn}, nil
}

func (c *cdnsService) Delete(id string) error {
	_, err := c.client.CDNs.Delete(context.TODO(), id)

	return err
}

func (c *cdnsService) FlushCache(id string, req *godo.CDNFlushCacheRequest) error {
	_, err := c.client.CDNs.FlushCache(context.TODO(), id, req)

	return err
}
