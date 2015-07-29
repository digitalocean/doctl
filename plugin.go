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

	"github.com/Sirupsen/logrus"
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

func (p *plugin) Exec(port string) error {
	path := filepath.Join(p.path, p.bin)
	cmd := exec.Command(path, "-port", port)
	logrus.WithFields(logrus.Fields{
		"options": fmt.Sprintf("%#v", cmd.Args),
	}).Debug("starting plugin")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Plugin lists all available plugins.
func Plugin(c *cli.Context) {
	if c.Args().Present() {
		execPlugin(c.Args().First(), c.Args().Tail())
		return
	}

	plugins := loadPlugins()

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

func execPlugin(name string, args []string) {
	logrus.Debug("execPlugin")
	logrus.WithFields(logrus.Fields{
		"name": name,
		"args": fmt.Sprintf("%#v", args)}).Debug("execing plugin with options")

	server := NewServer()
	go server.Serve()
	logrus.Debug("starting server")

	<-server.ready

	var pl plugin
	for _, p := range loadPlugins() {
		if p.name == name {
			pl = p
		}
	}

	if len(pl.name) < 1 {
		logrus.Fatalf("no plugin found: %s", name)
	}

	// exec plugin and get standard output
	err := pl.Exec(server.addr)
	if err != nil {
		logrus.WithField("err", err).Fatalf("could not execute plugin")
	}

	server.Stop()

}
