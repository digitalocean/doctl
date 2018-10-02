/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pluginhost

import (
	"fmt"
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"

	"github.com/digitalocean/doctl"
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
		AccessToken: viper.GetString(doctl.ArgAccessToken),
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
