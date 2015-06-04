package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/gopkg.in/yaml.v1"
)

type CLIOutput struct {
	w *tabwriter.Writer
}

func NewCLIOutput() *CLIOutput {
	return &CLIOutput{
		w: tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0),
	}
}

func WriteOutput(data interface{}) {
	var output []byte
	var err error

	switch OutputFormat {
	case "json":
		output, err = json.Marshal(data)
		if err != nil {
			log.Fatalf("JSON Encoding Error: %s", err)
		}

	case "yaml":
		output, err = yaml.Marshal(data)
		if err != nil {
			log.Fatalf("YAML Encoding Error: %s", err)
		}
	}
	fmt.Println(string(output))
}

func (c *CLIOutput) Header(a ...string) {
	fmt.Fprintln(c.w, strings.Join(a, "\t"))
}

func (c *CLIOutput) Writeln(format string, a ...interface{}) {
	fmt.Fprintf(c.w, format, a...)
}

func (c *CLIOutput) Flush() {
	c.w.Flush()
}
