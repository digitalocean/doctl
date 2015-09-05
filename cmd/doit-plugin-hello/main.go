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
	pluginName = "hello"
)

type helloPlugin struct{}

var _ doit.Plugin = &helloPlugin{}

func (p *helloPlugin) Name() string {
	return pluginName
}

var (
	serverPort = flag.String("port", ":0", "RPC port")
)

var (
	summary = flag.Bool("summary", false, "show summary")
)

type server struct{}

// Execute
func (s server) Execute(ctx context.Context, in *protos.PluginRequest) (*protos.PluginReply, error) {
	return &protos.PluginReply{Output: fmt.Sprintf("hello: %+v", in)}, nil
}

func main() {
	flag.Parse()

	if *summary {
		fmt.Println("sample plugin")
		os.Exit(0)
	}

	p := &helloPlugin{}
	c, err := doit.NewPluginClient(p, *serverPort, &server{})
	if err != nil {
		log.Fatalf("error initializing plugin: %v", err)
	}

	defer c.Close()
	c.Serve()
}
