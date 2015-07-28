package doit

import (
	"net"

	"google.golang.org/grpc"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/protos"
	"golang.org/x/net/context"
)

// ServerRPC handles RPC For doit.
type ServerRPC struct {
	Addr string
}

// NewServerRPC creates a ServerRPC.
func NewServerRPC(addr string) *ServerRPC {
	return &ServerRPC{
		Addr: addr,
	}
}

// Register implements Register from the doit protobuf interface.
func (s ServerRPC) Register(ctx context.Context, in *protos.RegisterRequest) (*protos.RegisterReply, error) {
	return &protos.RegisterReply{Address: s.Addr}, nil
}

// Serve serves stuff.
func Serve() {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to open server port")
	}

	server := NewServerRPC(l.Addr().String())

	s := grpc.NewServer()
	protos.RegisterDoitServer(s, server)
	s.Serve(l)
}
