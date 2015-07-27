package doit

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/codegangsta/cli"
)

const (
	pluginPrefix = "doit-plugin-"
)

var (
	pluginPattern = fmt.Sprintf("^%s[A-Za-z0-9\\-]+$", pluginPrefix)
	pluginNameRE  = regexp.MustCompile(pluginPattern)
)

type plugin struct {
	name string
	path string
	bin  string
}

func loadPlugins() []plugin {
	plugins := []plugin{}

	for _, p := range pluginPaths() {
		files, err := ioutil.ReadDir(p)
		if err != nil {
			continue
		}

		for _, f := range files {
			if pluginNameRE.MatchString(f.Name()) {
				plugin := newPlugin(f.Name(), p)
				plugins = append(plugins, *plugin)
			}
		}
	}

	return plugins
}

func newPlugin(bin, path string) *plugin {
	name := strings.TrimPrefix(bin, "doit-plugin-")
	return &plugin{
		bin:  bin,
		name: name,
		path: path,
	}
}

func (p *plugin) Summary() (string, error) {
	path := filepath.Join(p.path, p.bin)
	out, err := exec.Command(path, "-summary").Output()
	return string(out), err
}

// Plugin lists all available plugins.
func Plugin(c *cli.Context) {
	if c.Args().Present() {
		fmt.Printf("name: %s, args: %#v\n", c.Args().First(), c.Args().Tail())
		return
	}

	plugins := loadPlugins()
	fmt.Printf("%#v\n", plugins)

	w := new(tabwriter.Writer)
	w.Init(c.App.Writer, 0, 8, 1, '\t', 0)

	fmt.Fprintln(w, "Plugin\tSummary")
	for _, p := range plugins {
		out, err := p.Summary()
		if err == nil {
			fmt.Fprintf(w, "%s\t%s", p.name, out)
		}
	}

	w.Flush()
}

func pluginPaths() []string {
	return []string{
		filepath.Join(os.Getenv("GOPATH"), "bin"),
	}
}
