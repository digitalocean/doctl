package doit

import (
	"bytes"
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

func (p *plugin) Exec() error {
	path := filepath.Join(p.path, p.bin)
	cmd := exec.Command(path)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		logrus.WithField("err", err).Fatal("could not start plugin")
	}

	var buffer bytes.Buffer
	for {
		var b []byte
		n, err := stdout.Read(b)
		if err != nil {
			logrus.WithField("err", err).Fatal("couldn't read input")
		}

		buffer.Write(b)
	}

	if err := cmd.Wait(); err != nil {
		logrus.WithField("err", err).Fatal("something went wrong")
	}

	return nil
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
	fmt.Printf("name: %s, args: %#v\n", name, args)

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
	err := pl.Exec()
	if err != nil {
		logrus.WithField("err", err).Fatalf("could not execute plugin")
	}

}
