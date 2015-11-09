package doit

import (
	"net/url"
	"strconv"

	log "github.com/bryanl/doit/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/viper"
)

// Generator is a function that generates the list to be paginated.
type Generator func(*godo.ListOptions) ([]interface{}, *godo.Response, error)

// PaginateResp paginates a Response.
func PaginateResp(gen Generator) ([]interface{}, error) {
	opt := &godo.ListOptions{Page: 1, PerPage: 200}
	list := []interface{}{}

	for {
		items, resp, err := gen(opt)
		if err != nil {
			return nil, err
		}

		for _, i := range items {
			list = append(list, i)
		}

		if resp == nil || resp.Links.Pages == nil {
			break
		}

		if uStr := resp.Links.Pages.Next; len(uStr) > 0 {
			u, err := url.Parse(uStr)
			if err != nil {
				return nil, err
			}

			if viper.GetBool("debug") {
				log.WithFields(log.Fields{
					"page.current": opt.Page,
					"page.per":     opt.PerPage,
				}).Info("retrieving page")
			}
			pageStr := u.Query().Get("page")
			page, err := strconv.Atoi(pageStr)
			if err != nil {
				return nil, err
			}

			opt.Page = page
			continue
		}

		break
	}

	return list, nil

}
