/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package displayers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"

	"github.com/digitalocean/doctl"
)

var (
	hc = &headerControl{}
)

func newTabWriter(out io.Writer) *tabwriter.Writer {
	w := new(tabwriter.Writer)
	w.Init(out, 0, 0, 4, ' ', 0)

	return w
}

type headerControl struct {
	hideHeader bool
}

func (hc *headerControl) HideHeader(hide bool) {
	hc.hideHeader = hide
}

func prettyPrintStruct(obj interface{}) string {
	output := []string{}

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovered from %v", err)
		}
	}()

	val := reflect.Indirect(reflect.ValueOf(obj))
	for i := 0; i < val.NumField(); i++ {
		k := strings.Split(val.Type().Field(i).Tag.Get("json"), ",")[0]
		v := reflect.ValueOf(val.Field(i).Interface())
		output = append(output, fmt.Sprintf("%v:%v", k, v))
	}

	return strings.Join(output, ",")
}

// Displayable is a displable entity. These are used for printing results.
type Displayable interface {
	Cols() []string
	ColMap() map[string]string
	KV() []map[string]interface{}
	JSON(io.Writer) error
}

type Displayer struct {
	NS     string
	Config doctl.Config
	Item   Displayable
	Out    io.Writer
}

func (d *Displayer) Display() error {
	output, err := doctl.DoitConfig.GetString(doctl.NSRoot, "output")
	if err != nil {
		return nil
	}

	if output == "" {
		output = "text"
	}

	switch output {
	case "json":
		return d.Item.JSON(d.Out)
	case "text":
		cols, err := handleColumns(d.NS, d.Config)
		if err != nil {
			return err
		}

		return displayText(d.Item, d.Out, cols)
	default:
		return fmt.Errorf("unknown output type")
	}
}

func writeJSON(item interface{}, w io.Writer) error {
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	err = json.Indent(&out, b, "", "  ")
	if err != nil {
		return err
	}
	_, err = out.WriteTo(w)

	return err
}

func displayText(item Displayable, out io.Writer, includeCols []string) error {
	w := newTabWriter(out)

	cols := item.Cols()
	if len(includeCols) > 0 && includeCols[0] != "" {
		cols = includeCols
	}

	if !hc.hideHeader {
		headers := []string{}
		for _, k := range cols {
			col := item.ColMap()[k]
			if col == "" {
				return fmt.Errorf("unknown column %q", k)
			}

			headers = append(headers, col)
		}
		fmt.Fprintln(w, strings.Join(headers, "\t"))
	}

	for _, r := range item.KV() {
		values := []interface{}{}
		formats := []string{}

		for _, col := range cols {
			v := r[col]

			values = append(values, v)

			switch v.(type) {
			case string:
				formats = append(formats, "%s")
			case int:
				formats = append(formats, "%d")
			case float64:
				formats = append(formats, "%f")
			case bool:
				formats = append(formats, "%v")
			default:
				formats = append(formats, "%v")
			}
		}
		format := strings.Join(formats, "\t")
		fmt.Fprintf(w, format+"\n", values...)
	}

	return w.Flush()
}
