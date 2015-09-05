package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/bryanl/doit/protos"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
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
	pluginFactory = func(path string) doit.Command {
		return doit.NewLiveCommand(path)
	}
	pluginLoader = func() []plugin { return loadPlugins() }
)

type plugin struct {
	name    string
	command doit.Command

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

// Plugin generates a plugin command.
func Plugin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "plugin commands",
		Long:  "plugin commands",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunPlugin(args, writer))
		},
	}

	return cmd
}

// RunPlugin lists all available plugins.
func RunPlugin(args []string, out io.Writer) error {
	if len(args) > 1 {
		execPlugin(args[0], args[1:])
		return nil
	}

	plugins := pluginLoader()

	w := new(tabwriter.Writer)
	w.Init(out, 0, 8, 1, '\t', 0)

	fmt.Fprintln(w, "Plugin\tSummary")
	for _, p := range plugins {
		out, err := p.Summary()
		if err == nil {
			fmt.Fprintf(w, "%s\t%s", p.name, out)
		}
	}

	w.Flush()

	return nil
}

func pluginPaths() []string {
	return defaultPluginPaths
}

func execPlugin(name string, args []string) {
	logrus.Debug("execPlugin")
	logrus.WithFields(logrus.Fields{
		"name": name,
		"args": fmt.Sprintf("%#v", args)}).Debug("execing plugin with options")

	server := doit.NewServer()
	go server.Serve()
	logrus.Debug("starting server")

	<-server.Ready
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
	go pl.Exec(server.Addr)

	logrus.Debug("waiting for server to be ready")
	<-server.Ready

	logrus.Debugf("ready to go? %#v", server)

	conn, err := grpc.Dial(server.Remote)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not connect to server")
	}

	defer conn.Close()

	c := protos.NewPluginClient(conn)

	o := []*protos.PluginRequest_Option{}
	if as := argSlicer(args); len(as) > 1 {
		for _, a := range as {
			o1 := &protos.PluginRequest_Option{
				Name:  a[0],
				Value: a[1],
			}
			o = append(o, o1)
		}
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
