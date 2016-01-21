package commands

import (
	"strings"

	"github.com/bryanl/doit"
)

func handleColumns(ns string, config doit.Config) ([]string, error) {
	colStr, err := config.GetString(ns, doit.ArgFormat)
	if err != nil {
		return nil, err
	}

	var cols []string
	for _, c := range strings.Split(strings.Join(strings.Fields(colStr), ""), ",") {
		if c != "" {
			cols = append(cols, c)
		}
	}

	hh, err := config.GetBool(ns, doit.ArgNoHeader)
	if err != nil {
		return nil, err
	}

	hc.HideHeader(hh)

	return cols, nil
}
