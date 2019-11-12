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

package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	k8sapiv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

type dockerConfig struct {
	Auths map[string]struct {
		Auth string `json:"auth,omitempty"`
	} `json:"auths"`
}

// Registry creates the registry command
func Registry() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "registry",
			Aliases: []string{"reg", "r"},
			Short:   "registry commands",
			Long:    "registry is used to access container registry commands",
		},
	}

	CmdBuilder(cmd, RunRegistryCreate, "create <registry-name>", "create container registry", Writer)

	CmdBuilder(cmd, RunRegistryGet, "get", "get the container registry", Writer, aliasOpt("g"), displayerType(&displayers.Registry{}))

	cmdRunRegistryDelete := CmdBuilder(cmd, RunRegistryDelete, "delete", "delete the container registry", Writer, aliasOpt("del"))
	AddBoolFlag(cmdRunRegistryDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force registry delete")

	CmdBuilder(cmd, RunRegistryLogin, "login", "log in Docker to the container registry", Writer)

	cmdRunKubernetesManifest := CmdBuilder(cmd, RunKubernetesManifest, "kubernetes-manifest", "create a Kubernetes secret manifest to allow access to the registry", Writer, aliasOpt("k8s"))
	AddStringFlag(cmdRunKubernetesManifest, doctl.ArgObjectName, "", "", "the secret name to create. defaults to the registry name prefixed with \"registry-\"")
	AddStringFlag(cmdRunKubernetesManifest, doctl.ArgObjectNamespace, "", "default", "the namespace to hold the secret")

	return cmd
}

// Registry

// RunRegistryCreate creates a registry
func RunRegistryCreate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]
	rs := c.Registry()

	rcr := &godo.RegistryCreateRequest{Name: name}
	r, err := rs.Create(rcr)
	if err != nil {
		return err
	}

	return displayRegistries(c, *r)
}

// RunRegistryGet returns the registry
func RunRegistryGet(c *CmdConfig) error {
	reg, err := c.Registry().Get()
	if err != nil {
		return err
	}

	return displayRegistries(c, *reg)
}

// RunRegistryDelete delete the registry
func RunRegistryDelete(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirm("delete registry") != nil {
		return fmt.Errorf("operation aborted")
	}

	return c.Registry().Delete()
}

// DockerConfigProvider allows a user to read from a remote and local Kubeconfig, and write to a
// local Kubeconfig.
type DockerConfigProvider interface {
	ConfigPath() string
}

// RunRegistryLogin logs in Docker to the registry
func RunRegistryLogin(c *CmdConfig) error {
	conf, err := c.Registry().DockerCredentials()
	if err != nil {
		return err
	}

	dc := &dockerConfig{}
	err = json.Unmarshal(conf, dc)
	if err != nil {
		return err
	}

	// read the login credentials from the docker config
	for host, conf := range dc.Auths {
		// decode and split into username + password
		creds, err := base64.StdEncoding.DecodeString(conf.Auth)
		if err != nil {
			return err
		}

		splitCreds := strings.Split(string(creds), ":")
		if len(splitCreds) != 2 {
			return fmt.Errorf("got invalid docker credentials")
		}
		user, pass := splitCreds[0], splitCreds[1]

		// log in via the docker cli
		args := []string{
			"login", host,
			"-u", user,
			"--password-stdin",
		}
		cmd := exec.Command("docker", args...)
		cmd.Stdin = strings.NewReader(pass)
		cmd.Stdout = c.Out
		cmd.Stderr = c.Out

		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	fmt.Println("logged in to the registry")

	return nil
}

// RunKubernetesManifest prints a Kubernetes manifest that provides access to the registry
func RunKubernetesManifest(c *CmdConfig) error {
	secretName, err := c.Doit.GetString(c.NS, doctl.ArgObjectName)
	if err != nil {
		return err
	}
	secretNamespace, err := c.Doit.GetString(c.NS, doctl.ArgObjectNamespace)
	if err != nil {
		return err
	}

	// if no secret name supplied, use the registry name
	if secretName == "" {
		reg, err := c.Registry().Get()
		if err != nil {
			return err
		}
		secretName = reg.Name
	}

	// fetch docker config
	dockerConfig, err := c.Registry().DockerCredentials()
	if err != nil {
		return err
	}

	// create the manifest for the secret
	secret := &k8sapiv1.Secret{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      "registry-" + secretName,
			Namespace: secretNamespace,
		},
		Type: k8sapiv1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": dockerConfig,
		},
	}

	serializer := k8sjson.NewSerializerWithOptions(
		k8sjson.DefaultMetaFactory, nil, nil,
		k8sjson.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)

	return serializer.Encode(secret, c.Out)
}

func displayRegistries(c *CmdConfig, registries ...do.Registry) error {
	item := &displayers.Registry{
		Registries: registries,
	}
	return c.Display(item)
}
