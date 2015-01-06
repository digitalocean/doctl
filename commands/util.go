package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v1"
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
			fmt.Printf("JSON Encoding Error: %s", err)
			os.Exit(1)
		}

	case "yaml":
		output, err = yaml.Marshal(data)
		if err != nil {
			fmt.Printf("YAML Encoding Error: %s", err)
			os.Exit(1)
		}
	}
	fmt.Printf("%s", string(output))
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
