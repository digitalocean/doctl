package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/protos"
	"google.golang.org/grpc"
)

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

	l, err := net.Listen("tcp", *serverPort)
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to open port")
	}

	fmt.Printf("%s", l.Addr().String())

	s := grpc.NewServer()
	protos.RegisterPluginServer(s, &server{})
	s.Serve(l)
}
