package doit

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"text/tabwriter"

	"github.com/codegangsta/cli"
)

var (
	pluginNameRE = regexp.MustCompile("^doit-plugin-[A-Za-z0-9\\-]+$")
)

// PluginList lists all available plugins.
func PluginList(c *cli.Context) {
	w := new(tabwriter.Writer)
	w.Init(c.App.Writer, 0, 8, 1, '\t', 0)

	fmt.Fprintln(w, "Plugin\tSummary")

	for _, p := range pluginPaths() {
		files, _ := ioutil.ReadDir(p)
		for _, f := range files {
			if pluginNameRE.MatchString(f.Name()) {
				bin := filepath.Join(p, f.Name())
				out, err := exec.Command(bin, "-summary").Output()
				if err == nil {
					fmt.Fprintf(w, "%s\t%s", f.Name(), string(out))
				}
			}
		}
	}

	w.Flush()
}

func pluginPaths() []string {
	return []string{
		filepath.Join(os.Getenv("GOPATH"), "bin"),
	}
}
