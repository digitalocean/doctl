package doit

import (
	"errors"
	"fmt"
	"net"

	"github.com/bryanl/doit/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Plugin is a a plugin.
type Plugin interface {
	Name() string
}

// PluginClient manages the client connection back to the doit rpc server.
type PluginClient struct {
	conn     *grpc.ClientConn
	server   protos.PluginServer
	listener net.Listener
}

// NewPluginClient creates a PluginClient.
func NewPluginClient(p Plugin, sp string, server protos.PluginServer) (*PluginClient, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(sp)
	if err != nil {
		return nil, err
	}

	c := protos.NewDoitClient(conn)

	req := &protos.RegisterRequest{
		Name:    p.Name(),
		Address: l.Addr().String(),
	}

	reply, err := c.Register(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("unable to register server: %v", err)
	}

	if !reply.Success {
		return nil, errors.New("unable to register plugin")
	}

	return &PluginClient{
		conn:     conn,
		server:   server,
		listener: l,
	}, nil
}

// Close closes the client connection to the rpc server.
func (pc *PluginClient) Close() error {
	return pc.Close()
}

// Serve starts the plugin client listener.
func (pc *PluginClient) Serve() error {
	s := grpc.NewServer()
	protos.RegisterPluginServer(s, pc.server)
	return s.Serve(pc.listener)
}
