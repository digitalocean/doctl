package main

import (
	"fmt"
	"os"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/protos"
	"google.golang.org/grpc"
)

func main() {
	address := os.Args[1]
	conn, err := grpc.Dial(address)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not connect to server")
	}

	defer conn.Close()

	c := protos.NewPluginClient(conn)

	o := []*protos.PluginRequest_Option{
		{Name: "option1", Value: "hello"},
		{Name: "option2", Value: "yay!"},
	}
	r, err := c.Execute(context.Background(), &protos.PluginRequest{Option: o})
	if err != nil {
		logrus.WithField("err", err).Fatal("could not execute")
	}

	fmt.Println("Output:", r.Output)
}
