package pie_test

import (
	"log"
	"os"
	"strings"

	"net/rpc/jsonrpc"

	"github.com/natefinch/pie"
)

// This function should be called from the master program that wants to run
// plugins to extend its functionality.
//
// This example shows the master program starting a plugin at path
// "/var/lib/foo", using JSON-RPC, and writing its output to this application's
// Stderr.  The application can then call methods on the rpc client returned
// using the standard rpc pattern.
func ExampleStartProviderCodec() {
	client, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, os.Stderr, "/var/lib/foo")
	if err != nil {
		log.Fatalf("failed to load foo plugin: %s", err)
	}
	var reply string
	client.Call("Foo.ToUpper", "something", &reply)
}

// This function should be called from the plugin program that wants to provide
// functionality for the master program.
//
// This example shows the plugin starting a JSON-RPC server to be accessed by
// the master program. Server.ServeCodec() will block forever, so it is common
// to simply put this at the end of the plugin's main function.
func ExampleProvider_ServeCodec() {
	p := pie.NewProvider()
	if err := p.RegisterName("Foo", API{}); err != nil {
		log.Fatalf("can't register api: %s", err)
	}
	p.ServeCodec(jsonrpc.NewServerCodec)
}

// This function should be called from the plugin program that wants to consume
// an API from the master program.
//
// This example shows the plugin creating a JSON-RPC client talks to the host
// application.
func ExampleNewConsumerCodec() {
	client := pie.NewConsumerCodec(jsonrpc.NewClientCodec)
	var reply string
	client.Call("Foo.ToUpper", "something", &reply)
}

// API is an example type to show how to serve methods over RPC.
type API struct{}

// ToUpper is an example function that gets served over RPC.  See net/rpc for
// details on how to server functionality over RPC.
func (API) ToUpper(input string, output *string) error {
	*output = strings.ToUpper(input)
	return nil
}
