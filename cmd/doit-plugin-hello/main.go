package main

import (
	"fmt"
	"net"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/protos"
	"google.golang.org/grpc"
)

const (
	serverPort = ":0"
)

type server struct{}

// SayHello implements helloworld.GreeterServer
func (s server) Execute(ctx context.Context, in *protos.PluginRequest) (*protos.PluginReply, error) {
	return &protos.PluginReply{Output: fmt.Sprintf("hello: %+v", in)}, nil
}

func main() {
	l, err := net.Listen("tcp", serverPort)
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to open port")
	}

	fmt.Printf("%s", l.Addr().String())

	s := grpc.NewServer()
	protos.RegisterPluginServer(s, &server{})
	s.Serve(l)
}
