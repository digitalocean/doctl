package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bryanl/doit"
)

type displayer interface {
	Cols() []string
	ColMap() map[string]string
	KV() []map[string]interface{}
	JSON(io.Writer) error
}

// displayOutput displays an object or group of objects to a user. It
// checks to see what the output type should be.
func displayOutput(item displayer, out io.Writer) error {
	output, err := doit.DoitConfig.GetString(doit.NSRoot, "output")
	if err != nil {
		return nil
	}

	if output == "" {
		output = "text"
	}

	switch output {
	case "json":
		return item.JSON(out)
	case "text":
		return displayText(item, out)
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

func displayText(item displayer, out io.Writer) error {
	w := newTabWriter(out)

	cols := item.Cols()
	headers := make([]string, len(cols))
	for i, k := range cols {
		headers[i] = item.ColMap()[k]
	}

	fmt.Fprintln(w, strings.Join(headers, "\t"))

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
