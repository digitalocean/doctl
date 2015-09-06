package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/net/context"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/protos"
)

const (
	pluginName        = "setup"
	pluginDescription = "setup clusters, because why not?"
)

var (
	serverPort = flag.String("port", ":0", "RPC port")
	summary    = flag.Bool("summary", false, "show summary")
)

type setupPlugin struct {
}

type server struct{}

// Execute
func (s server) Execute(ctx context.Context, in *protos.PluginRequest) (*protos.PluginReply, error) {
	return &protos.PluginReply{Output: "stub output for future setup plugin"}, nil
}

var _ doit.Plugin = &setupPlugin{}

func (p *setupPlugin) Name() string {
	return pluginName
}

func (p *setupPlugin) Description() string {
	return pluginDescription
}

func main() {
	flag.Parse()

	p := &setupPlugin{}

	if *summary {
		fmt.Println(p.Description())
		os.Exit(0)
	}

	c, err := doit.NewPluginClient(p, *serverPort, &server{})
	if err != nil {
		log.Fatalf("error initializing plugin: %v", err)
	}

	defer c.Close()
	c.Serve()
}
