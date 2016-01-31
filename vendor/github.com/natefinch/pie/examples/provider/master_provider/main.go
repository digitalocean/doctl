// Command example_master is a simple example of a master application that runs
// a standard provider plugin.
//
// It communicates with the plugin using JSON-RPC. It expects example_plugin to
// be in the same directory as itself, and will start it when it runs.
package main

import (
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"runtime"

	"github.com/natefinch/pie"
)

func main() {
	log.SetPrefix("[master log] ")

	path := "plugin_provider"
	if runtime.GOOS == "windows" {
		path = path + ".exe"
	}

	client, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, os.Stderr, path)
	if err != nil {
		log.Fatalf("Error running plugin: %s", err)
	}
	defer client.Close()
	p := plug{client}
	res, err := p.SayHi("master")
	if err != nil {
		log.Fatalf("error calling SayHi: %s", err)
	}
	log.Printf("Response from plugin: %q", res)

	res, err = p.SayBye("master")
	if err != nil {
		log.Fatalf("error calling SayBye: %s", err)
	}
	log.Printf("Response from plugin2: %q", res)

}

type plug struct {
	client *rpc.Client
}

func (p plug) SayHi(name string) (result string, err error) {
	err = p.client.Call("Plugin.SayHi", name, &result)
	return result, err
}

func (p plug) SayBye(name string) (result string, err error) {
	err = p.client.Call("Plugin2.SayBye", name, &result)
	return result, err
}
