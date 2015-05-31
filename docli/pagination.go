package docli

import (
	"fmt"
	"net/url"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/digitalocean/godo"
)

// Generator is a function that generates the list to be paginated.
type Generator func(*godo.ListOptions) ([]interface{}, *godo.Response, error)

// PaginateResp paginates a Response.
func PaginateResp(gen Generator, opts *Opts) ([]interface{}, error) {
	opt := &godo.ListOptions{}
	list := []interface{}{}
	fmt.Printf("opts: %#v\n", opts)

	for {
		log.WithFields(log.Fields{
			"page":     opt.Page,
			"per_page": opt.PerPage,
		}).Warn("the opts")

		items, resp, err := gen(opt)
		if err != nil {
			return nil, err
		}

		log.WithFields(log.Fields{
			"pages": fmt.Sprintf("%#v", resp.Links.Pages),
			"resp":  fmt.Sprintf("%#v", resp.Links),
		}).Warn("current page")

		for _, i := range items {
			list = append(list, i)
		}

		if uStr := resp.Links.Pages.Next; len(uStr) > 0 {
			u, err := url.Parse(uStr)
			if err != nil {
				return nil, err
			}

			log.WithFields(log.Fields{
				"next_url": u.String(),
			}).Warn("current page")

			pageStr := u.Query().Get("page")
			page, err := strconv.Atoi(pageStr)
			if err != nil {
				return nil, err
			}

			opt.Page = page
			continue
		}

		break

		//if resp.Links == nil || resp.Links.IsLastPage() {
		//break
		//}

		//page, err := resp.Links.CurrentPage()
		//if err != nil {
		//return nil, err
		//}

		//opt.Page = page + 1
	}

	return list, nil

}
