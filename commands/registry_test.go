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
	"errors"
	"os"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	k8sapiv1 "k8s.io/api/core/v1"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	testRegistryName  = "container-registry"
	testRegistry      = do.Registry{Registry: &godo.Registry{Name: testRegistryName}}
	testRepoName      = "test-repository"
	testRepositoryTag = do.RepositoryTag{
		RepositoryTag: &godo.RepositoryTag{
			RegistryName:        testRegistryName,
			Repository:          testRepoName,
			Tag:                 "tag",
			ManifestDigest:      "sha256:6c3c624b58dbbcd3c0dd82b4c53f04194d1247c6eebdaab7c610cf7d66709b3b",
			CompressedSizeBytes: 50,
			SizeBytes:           100,
			UpdatedAt:           time.Now(),
		},
	}
	testRepository = do.Repository{
		Repository: &godo.Repository{
			RegistryName: testRegistryName,
			Name:         testRegistryName,
			TagCount:     5,
			LatestTag:    testRepositoryTag.RepositoryTag,
		},
	}
	testDockerCredentials = &godo.DockerCredentials{
		// the base64 string is "username:password"
		DockerConfigJSON: []byte(`{"auths":{"hostname":{"auth":"dXNlcm5hbWU6cGFzc3dvcmQ="}}}`),
	}
)

func TestRegistryCommand(t *testing.T) {
	cmd := Registry()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "get", "delete", "login", "logout", "kubernetes-manifest", "repository", "docker-config")
}

func TestRepositoryCommand(t *testing.T) {
	cmd := Repository()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list", "list-tags", "delete-manifest", "delete-tag")
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

func TestDockerConfig(t *testing.T) {
	tests := []struct {
		name      string
		readWrite bool
	}{
		{
			name:      "read-only",
			readWrite: false,
		},
		{
			name:      "read-write",
			readWrite: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.registry.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
					ReadWrite: test.readWrite,
				}).Return(testDockerCredentials, nil)

				config.Doit.Set(config.NS, doctl.ArgReadWrite, test.readWrite)

				var output bytes.Buffer
				config.Out = &output

				err := RunDockerConfig(config)
				assert.NoError(t, err)

				expectedOutput := append(testDockerCredentials.DockerConfigJSON, '\n')
				assert.Equal(t, expectedOutput, output.Bytes())
			})
		})
	}
}

func TestRepositoryList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Get().Return(&testRegistry, nil)
		tm.registry.EXPECT().ListRepositories(testRepository.RegistryName).Return([]do.Repository{testRepository}, nil)

		err := RunListRepositories(config)
		assert.NoError(t, err)
	})
}

func TestRepositoryListTags(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Get().Return(&testRegistry, nil)
		tm.registry.EXPECT().ListRepositoryTags(
			testRepositoryTag.RegistryName,
			testRepositoryTag.Repository,
		).Return([]do.RepositoryTag{testRepositoryTag}, nil)
		config.Args = append(config.Args, testRepositoryTag.Repository)

		err := RunListRepositoryTags(config)
		assert.NoError(t, err)
	})
}

func TestRepositoryDeleteTag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expect      func(m *mocks.MockRegistryService)
		expectedErr string
	}{
		{
			name: "no deletion arguments",
			args: []string{
				testRepositoryTag.Repository,
			},
			expectedErr: "(test) command is missing required arguments",
		},
		{
			name: "one tag, successful",
			args: []string{
				testRepositoryTag.Repository,
				testRepositoryTag.Tag,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Get().Return(&testRegistry, nil)
				m.EXPECT().DeleteTag(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					testRepositoryTag.Tag,
				).Return(nil)
			},
		},
		{
			name: "multiple tags, successful",
			args: []string{
				testRepositoryTag.Repository,
				testRepositoryTag.Tag,
				"extra-tag",
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Get().Return(&testRegistry, nil)
				m.EXPECT().DeleteTag(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					testRepositoryTag.Tag,
				).Return(nil)
				m.EXPECT().DeleteTag(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					"extra-tag",
				).Return(nil)
			},
		},
		{
			name: "multiple tags, partial failure",
			args: []string{
				testRepositoryTag.Repository,
				"fail-tag",
				testRepositoryTag.Tag,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Get().Return(&testRegistry, nil)
				m.EXPECT().DeleteTag(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					"fail-tag",
				).Return(errors.New("oops"))
				m.EXPECT().DeleteTag(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					testRepositoryTag.Tag,
				).Return(nil)
			},
			expectedErr: "failed to delete all repository tags: \noops",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if test.expect != nil {
					test.expect(tm.registry)
				}

				config.Doit.Set(config.NS, doctl.ArgForce, true)
				config.Args = append(config.Args, test.args...)

				err := RunRepositoryDeleteTag(config)
				if test.expectedErr == "" {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
					assert.Equal(t, test.expectedErr, err.Error())
				}
			})
		})
	}
}

func TestRepositoryDeleteManifest(t *testing.T) {
	extraDigest := "sha256:5b0bcabd1ed22e9fb1310cf6c2dec7cdef19f0ad69efa1f392e94a4333501270"
	tests := []struct {
		name        string
		args        []string
		expect      func(m *mocks.MockRegistryService)
		expectedErr string
	}{
		{
			name: "no deletion arguments",
			args: []string{
				testRepositoryTag.Repository,
			},
			expectedErr: "(test) command is missing required arguments",
		},
		{
			name: "one digest, successful",
			args: []string{
				testRepositoryTag.Repository,
				testRepositoryTag.ManifestDigest,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Get().Return(&testRegistry, nil)
				m.EXPECT().DeleteManifest(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					testRepositoryTag.ManifestDigest,
				).Return(nil)
			},
		},
		{
			name: "multiple digests, successful",
			args: []string{
				testRepositoryTag.Repository,
				testRepositoryTag.ManifestDigest,
				extraDigest,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Get().Return(&testRegistry, nil)
				m.EXPECT().DeleteManifest(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					testRepositoryTag.ManifestDigest,
				).Return(nil)
				m.EXPECT().DeleteManifest(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					extraDigest,
				).Return(nil)
			},
		},
		{
			name: "multiple digests, partial failure",
			args: []string{
				testRepositoryTag.Repository,
				"fail-digest",
				testRepositoryTag.ManifestDigest,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Get().Return(&testRegistry, nil)
				m.EXPECT().DeleteManifest(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					"fail-digest",
				).Return(errors.New("oops"))
				m.EXPECT().DeleteManifest(
					testRepositoryTag.RegistryName,
					testRepositoryTag.Repository,
					testRepositoryTag.ManifestDigest,
				).Return(nil)
			},
			expectedErr: "failed to delete all repository manifests: \noops",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if test.expect != nil {
					test.expect(tm.registry)
				}

				config.Doit.Set(config.NS, doctl.ArgForce, true)
				config.Args = append(config.Args, test.args...)

				err := RunRepositoryDeleteManifest(config)
				if test.expectedErr == "" {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
					assert.Equal(t, test.expectedErr, err.Error())
				}
			})
		})
	}
}

func TestRegistryKubernetesManifest(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// test cases
		tcs := []struct {
			argName, argNamespace           string
			expectedName, expectedNamespace string
		}{
			{
				argName:           "",
				argNamespace:      "default",
				expectedName:      "registry-" + testRegistry.Name,
				expectedNamespace: "default",
			},
			{
				argName:           "my-registry",
				argNamespace:      "secret-namespace",
				expectedName:      "my-registry",
				expectedNamespace: "secret-namespace",
			},
		}

		// mocks shared across both test cases
		// Get should be called only when a name isn't supplied, to look up the registry's name
		tm.registry.EXPECT().Get().Return(&testRegistry, nil).Times(1)
		// DockerCredentials should be called both times to retrieve the credentials
		tm.registry.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
			ReadWrite: false,
		}).Return(testDockerCredentials, nil).Times(2)

		// tests
		for _, tc := range tcs {
			config.Doit.Set(config.NS, doctl.ArgObjectName, tc.argName)
			config.Doit.Set(config.NS, doctl.ArgObjectNamespace, tc.argNamespace)

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
			assert.Equal(t, tc.expectedName, secret.ObjectMeta.Name)
			assert.Equal(t, tc.expectedNamespace, secret.ObjectMeta.Namespace)
			assert.Contains(t, secret.Data, ".dockerconfigjson")
			assert.Equal(t, secret.Data[".dockerconfigjson"], testDockerCredentials.DockerConfigJSON)
		}
	})
}

func TestRegistryLogin(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Endpoint().Return(do.RegistryHostname)
		tm.registry.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
			ReadWrite: true,
		}).Return(testDockerCredentials, nil)

		config.Out = os.Stderr
		err := RunRegistryLogin(config)
		assert.NoError(t, err)
	})
}

func TestRegistryLogout(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Endpoint().Return(do.RegistryHostname)

		config.Out = os.Stderr
		err := RunRegistryLogout(config)
		assert.NoError(t, err)
	})
}
