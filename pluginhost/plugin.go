package pluginhost

import (
	"fmt"
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"

	"github.com/natefinch/pie"
	"github.com/spf13/viper"
)

// Host is an object consumers can retrieve doit information from.
type Host struct {
	client *rpc.Client
}

// NewHost builds an instance of Host.
func NewHost(pluginPath string) (*Host, error) {
	client, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, os.Stderr, pluginPath)
	if err != nil {
		return nil, err
	}

	return &Host{
		client: client,
	}, nil
}

// Call a method on the plugin.
func (h *Host) Call(method string, args ...string) (string, error) {
	opts := &CallOptions{
		AccessToken: viper.GetString("access-token"),
		Args:        args,
	}

	var result string
	err := h.client.Call(method, opts, &result)
	if err != nil {
		debug(err.Error())
		return "", fmt.Errorf("unable to run plugin action %s", method)
	}

	return result, nil
}

func debug(msg string) {
	//if viper.GetBool("verbose") {
	log.Println(msg)
	//}
}

// CallOptions are options to a plugin call. This is exported so go based plugins
// can use the type.
type CallOptions struct {
	AccessToken string
	Args        []string
}
