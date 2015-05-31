package domainrecs

import (
	"github.com/bryanl/docli/docli"
	"github.com/digitalocean/godo"
)

// List records for a domain.
func List(client *godo.Client, opts *docli.Opts, domain string) ([]godo.DomainRecord, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Domains.Records(domain, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := docli.PaginateResp(f, opts)
	if err != nil {
		return nil, err
	}

	list := make([]godo.DomainRecord, len(si))
	for i := range si {
		list[i] = si[i].(godo.DomainRecord)
	}

	return list, nil
}
