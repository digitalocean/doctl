package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
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

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to open port")
	}

	fmt.Printf("%s", l.Addr().String())

	req := &protos.RegisterRequest{
		Name:    pluginName,
		Address: l.Addr().String(),
	}

	d := &doit.ProtobufDoitRPC{}

	reply, err = d.Register(context.Background(), req)
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to register server location")
	}

	if !reply.Success {
		logrus.Fatal("unable to successfully register plugin")
	}

	s := grpc.NewServer()
	protos.RegisterPluginServer(s, &server{})
	s.Serve(l)
}
