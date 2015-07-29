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
	server *Server
}

// NewProtobufDoitRPC creates a ServerRPC.
func NewProtobufDoitRPC(server *Server) *ProtobufDoitRPC {
	return &ProtobufDoitRPC{server: server}
}

// Register implements Register from the doit protobuf interface.
func (s ProtobufDoitRPC) Register(ctx context.Context, in *protos.RegisterRequest) (*protos.RegisterReply, error) {
	logrus.WithField("incoming", fmt.Sprintf("%#v", in)).Debug("plugin registered")
	s.server.remote = in.Address
	s.server.ready <- true
	return &protos.RegisterReply{Success: true}, nil
}

// Server is our how doit serves.
type Server struct {
	addr       string
	remote     string
	server     *grpc.Server
	ready      chan bool
	registered chan bool
}

func NewServer() *Server {
	return &Server{
		ready:      make(chan bool, 1),
		registered: make(chan bool, 1),
	}
}

// Serve serves stuff.
func (server *Server) Serve() {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to open server port")
	}

	server.addr = l.Addr().String()

	server.ready <- true

	doitrpc := NewProtobufDoitRPC(server)

	server.server = grpc.NewServer()
	protos.RegisterDoitServer(server.server, doitrpc)
	server.server.Serve(l)
}

// Stop stops serving.
func (server *Server) Stop() {
	if server.server != nil {
		logrus.Debug("stopping server")
		server.server.Stop()
	}
}
