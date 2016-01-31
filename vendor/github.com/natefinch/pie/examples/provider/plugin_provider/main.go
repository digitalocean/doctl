// Command example_plugin is an example of a very simple plugin.
//
// example_plugin provides two APIs that communicate via JSON-RPC.  It is
// expected to be started by example_master.
package main

import (
	"log"
	"net/rpc/jsonrpc"

	"github.com/natefinch/pie"
)

func main() {
	log.SetPrefix("[plugin log] ")

	p := pie.NewProvider()
	if err := p.RegisterName("Plugin", api{}); err != nil {
		log.Fatalf("failed to register Plugin: %s", err)
	}
	if err := p.RegisterName("Plugin2", api2{}); err != nil {
		log.Fatalf("failed to register Plugin2: %s", err)
	}
	p.ServeCodec(jsonrpc.NewServerCodec)
}

type api struct{}

func (api) SayHi(name string, response *string) error {
	log.Printf("got call for SayHi with name %q", name)

	*response = "Hi " + name
	return nil
}

type api2 struct{}

func (api2) SayBye(name string, response *string) error {
	log.Printf("got call for SayBye with name %q", name)

	*response = "Bye " + name
	return nil
}
