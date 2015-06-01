package docli

import (
	"fmt"
	"net/url"
	"reflect"
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

	for {
		items, resp, err := gen(opt)
		if err != nil {
			return nil, err
		}

		for _, i := range items {
			list = append(list, i)
		}

		if uStr := resp.Links.Pages.Next; len(uStr) > 0 {
			u, err := url.Parse(uStr)
			if err != nil {
				return nil, err
			}

			if opts.Debug {
				log.WithFields(log.Fields{
					"page.current": opt.Page,
					"page.per":     opt.PerPage,
				}).Debug("retrieving page")
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

func PaginageResp2(f interface{}, opts *Opts) (interface{}, error) {
	opt := &godo.ListOptions{}
	vopt := reflect.ValueOf(opt)

	vf := reflect.ValueOf(f)
	fmt.Printf("vf: %#v\n", vf.Type().Out(0).Kind())

	vtype := vf.Type().Out(0)

	list := reflect.MakeSlice(reflect.SliceOf(vtype), 0, 1000)

	for {
		values := vf.Call([]reflect.Value{vopt})

		err := reflect.ValueOf(values[2]).Interface()

		switch e := err.(type) {
		case error:
			return nil, e
		}

		items := reflect.ValueOf(values[0])

		for i := 0; i < items.tLen(); i++ {
			list = reflect.Append(list, items.Index(i))
		}

		resp := reflect.ValueOf(values[1]).Interface().(*godo.Response)

		if uStr := resp.Links.Pages.Next; len(uStr) > 0 {
			u, err := url.Parse(uStr)
			if err != nil {
				return nil, err
			}

			if opts.Debug {
				log.WithFields(log.Fields{
					"page.current": opt.Page,
					"page.per":     opt.PerPage,
				}).Debug("retrieving page")
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

	return list.Interface(), fmt.Errorf("testing")
}
