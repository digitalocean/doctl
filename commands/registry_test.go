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
	"bytes"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	k8sapiv1 "k8s.io/api/core/v1"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	testRegistryName      = "container-registry"
	testRegistry          = do.Registry{Registry: &godo.Registry{Name: testRegistryName}}
	testDockerCredentials = []byte("valid docker config json")
)

func TestRegistryCommand(t *testing.T) {
	cmd := Registry()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "get", "delete", "login", "kubernetes-manifest")
}

func TestRegistryCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		rcr := godo.RegistryCreateRequest{Name: testRegistryName}
		tm.registry.EXPECT().Create(&rcr).Return(&testRegistry, nil)
		config.Args = append(config.Args, testRegistryName)

		err := RunRegistryCreate(config)
		assert.NoError(t, err)
	})
}

func TestRegistryGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Get().Return(&testRegistry, nil)

		err := RunRegistryGet(config)
		assert.NoError(t, err)
	})
}

func TestRegistryDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Delete().Return(nil)

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunRegistryDelete(config)
		assert.NoError(t, err)
	})
}

func TestRegistryKubernetesManifest(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Get().Return(&testRegistry, nil)
		tm.registry.EXPECT().DockerCredentials().Return(testDockerCredentials, nil)

		secretNamespace := "secret-namespace"
		config.Doit.Set(config.NS, doctl.ArgObjectNamespace, secretNamespace)

		var outputBuffer bytes.Buffer
		config.Out = &outputBuffer
		err := RunKubernetesManifest(config)
		assert.NoError(t, err)

		// check the object
		obj, _, err := k8sscheme.Codecs.UniversalDeserializer().Decode(outputBuffer.Bytes(), nil, nil)
		assert.NoError(t, err)
		secret := obj.(*k8sapiv1.Secret)

		assert.Equal(t, "Secret", secret.TypeMeta.Kind)
		assert.Equal(t, "v1", secret.TypeMeta.APIVersion)
		assert.Equal(t, k8sapiv1.SecretTypeDockerConfigJson, secret.Type)
		assert.Equal(t, "registry-"+testRegistry.Name, secret.ObjectMeta.Name)
		assert.Equal(t, secretNamespace, secret.ObjectMeta.Namespace)
		assert.Contains(t, secret.Data, ".dockerconfigjson")
		assert.Equal(t, secret.Data[".dockerconfigjson"], testDockerCredentials)
	})
}
