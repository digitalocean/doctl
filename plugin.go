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

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/protos"
	"github.com/codegangsta/cli"
)

const (
	pluginPrefix = "doit-plugin-"
)

var (
	pluginPattern      = fmt.Sprintf("^%s[A-Za-z0-9\\-]+$", pluginPrefix)
	pluginNameRE       = regexp.MustCompile(pluginPattern)
	defaultPluginPaths = []string{
		filepath.Join(os.Getenv("GOPATH"), "bin"),
	}
	pluginFactory = func(path string) Command {
		return NewLiveCommand(path)
	}
	pluginLoader = func() []plugin { return loadPlugins() }
)

type plugin struct {
	name    string
	command Command

	pluginCmd *exec.Cmd

	ready chan bool
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
	cmd := pluginFactory(filepath.Join(path, bin))
	return &plugin{
		name:    name,
		command: cmd,
		ready:   make(chan bool, 1),
	}
}

func (p *plugin) Summary() (string, error) {
	out, err := p.command.Run("-summary")
	return string(out), err
}

func (p *plugin) Exec(port string) error {
	return p.command.Start("-port", port)
}

func (p *plugin) Kill() error {
	return p.command.Stop()
}

// Plugin lists all available plugins.
func Plugin(c *cli.Context) {
	if c.Args().Present() {
		execPlugin(c.Args().First(), c.Args().Tail())
		return
	}

	plugins := pluginLoader()

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
	return defaultPluginPaths
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
	logrus.Debug("server ready")

	var pl plugin
	for _, p := range pluginLoader() {
		if p.name == name {
			pl = p
		}
	}

	if len(pl.name) < 1 {
		logrus.Fatalf("no plugin found: %s", name)
	}

	// exec plugin and get standard output
	go pl.Exec(server.addr)

	logrus.Debug("waiting for server to be ready")
	<-server.ready

	logrus.Debugf("ready to go? %#v", server)

	conn, err := grpc.Dial(server.remote)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not connect to server")
	}

	defer conn.Close()

	c := protos.NewPluginClient(conn)

	o := []*protos.PluginRequest_Option{}
	for _, a := range argSlicer(args) {
		o1 := &protos.PluginRequest_Option{
			Name:  a[0],
			Value: a[1],
		}
		o = append(o, o1)
	}

	r, err := c.Execute(context.Background(), &protos.PluginRequest{Option: o})
	if err != nil {
		logrus.WithField("err", err).Fatal("could not execute")
	}
	fmt.Println(r.Output)

	pl.Kill()

	server.Stop()
}

func argSlicer(args []string) [][]string {
	var c [][]string

	for _, a := range args {
		c = append(c, strings.Split(a, "="))
	}

	return c
}
