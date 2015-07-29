package doit

import (
	"net"

	"google.golang.org/grpc"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/protos"
	"golang.org/x/net/context"
)

// ProtobufDoitRPC handles RPC For doit.
type ProtobufDoitRPC struct{}

// NewProtobufDoitRPC creates a ServerRPC.
func NewProtobufDoitRPC() *ProtobufDoitRPC {
	return &ProtobufDoitRPC{}
}

// Register implements Register from the doit protobuf interface.
func (s ProtobufDoitRPC) Register(ctx context.Context, in *protos.RegisterRequest) (*protos.RegisterReply, error) {
	return &protos.RegisterReply{}, nil
}

// Server is our how doit serves.
type Server struct {
	addr   string
	server *grpc.Server
	ready  chan bool
}

func NewServer() *Server {
	return &Server{
		ready: make(chan bool, 1),
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

	doitrpc := NewProtobufDoitRPC()

	server.server = grpc.NewServer()
	protos.RegisterDoitServer(server.server, doitrpc)
	logrus.Warn("before")
	server.server.Serve(l)
	logrus.Warn("after")
}

// Stop stops serving.
func (server *Server) Stop() {
	if server.server != nil {
		server.server.Stop()
	}
}
