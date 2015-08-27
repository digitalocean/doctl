package doit

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/protos"
	"golang.org/x/net/context"
)

// ProtobufDoitRPC handles RPC For doit.
type ProtobufDoitRPC struct {
	Server *Server
}

// NewProtobufDoitRPC creates a ServerRPC.
func NewProtobufDoitRPC(server *Server) *ProtobufDoitRPC {
	return &ProtobufDoitRPC{Server: server}
}

// Register implements Register from the doit protobuf interface.
func (s ProtobufDoitRPC) Register(ctx context.Context, in *protos.RegisterRequest) (*protos.RegisterReply, error) {
	logrus.WithField("incoming", fmt.Sprintf("%#v", in)).Debug("plugin registered")
	s.Server.Remote = in.Address
	s.Server.Ready <- true
	return &protos.RegisterReply{Success: true}, nil
}

// Server is our how doit serves.
type Server struct {
	Addr       string
	Remote     string
	Server     *grpc.Server
	Ready      chan bool
	Registered chan bool
}

// NewServer creates an instance of Server.
func NewServer() *Server {
	return &Server{
		Ready:      make(chan bool, 1),
		Registered: make(chan bool, 1),
	}
}

// Serve serves stuff.
func (server *Server) Serve() {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to open server port")
	}

	server.Addr = l.Addr().String()

	server.Ready <- true

	doitrpc := NewProtobufDoitRPC(server)

	server.Server = grpc.NewServer()
	protos.RegisterDoitServer(server.Server, doitrpc)
	server.Server.Serve(l)
}

// Stop stops serving.
func (server *Server) Stop() {
	if server.Server != nil {
		logrus.Debug("stopping server")
		server.Server.Stop()
	}
}
