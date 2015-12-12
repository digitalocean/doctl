package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bryanl/doit"
)

type displayer interface {
	JSON(io.Writer) error
	String(io.Writer) error
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
		return item.String(out)
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
