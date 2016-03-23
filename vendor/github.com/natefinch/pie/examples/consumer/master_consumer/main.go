// Command example_host is an example of a main application that provides an API
// for plugins to consume.
//
// It expects to have the example_consumer application in the same directory,
// which it will start and then serve an API to it.
package main

import (
	"log"
	"os"
	"runtime"

	"github.com/natefinch/pie"
)

func main() {
	log.SetPrefix("[host log] ")
	path := "plugin_consumer"
	if runtime.GOOS == "windows" {
		path = path + ".exe"
	}

	s, err := pie.StartConsumer(os.Stderr, path)
	if err != nil {
		log.Fatalf("failed to start consumer: %s", err)
	}
	if err := s.RegisterName("Host", api{}); err != nil {
		log.Fatalf("failed to register Host: %s", err)
	}
	if err := s.RegisterName("Host2", api2{}); err != nil {
		log.Fatalf("failed to register Host2: %s", err)
	}
	s.Serve()
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
