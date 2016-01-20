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
	cols := strings.Split(colStr, ",")

	hh, err := config.GetBool(ns, doit.ArgNoHeader)
	if err != nil {
		return nil, err
	}

	hc.HideHeader(hh)

	return cols, nil
}
