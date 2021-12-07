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
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	k8sapiv1 "k8s.io/api/core/v1"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	testRegistryName     = "container-registry"
	testSubscriptionTier = "basic"
	invalidRegistryName  = "invalid-container-registry"
	testRegistry         = do.Registry{Registry: &godo.Registry{Name: testRegistryName}}
	testRepoName         = "test-repository"
	testRepositoryTag    = do.RepositoryTag{
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
	testRepositoryManifest = do.RepositoryManifest{
		RepositoryManifest: &godo.RepositoryManifest{
			RegistryName:        testRegistryName,
			Repository:          testRepoName,
			Digest:              "sha256:6c3c624b58dbbcd3c0dd82b4c53f04194d1247c6eebdaab7c610cf7d66709b3b",
			CompressedSizeBytes: 50,
			SizeBytes:           100,
			UpdatedAt:           time.Now(),
			Tags:                []string{"v1", "v2"},
			Blobs: []*godo.Blob{
				{
					Digest:              "sha256:123",
					CompressedSizeBytes: 123,
				},
				{
					Digest:              "sha256:456",
					CompressedSizeBytes: 456,
				},
			},
		},
	}
	testRepositoryManifestNoTags = do.RepositoryManifest{
		RepositoryManifest: &godo.RepositoryManifest{
			RegistryName:        testRegistryName,
			Repository:          testRepoName,
			Digest:              "sha256:6c3c624b58dbbcd3c0dd82b4c53f04194d1247c6eebdaab7c610cf7d66709b3b",
			CompressedSizeBytes: 50,
			SizeBytes:           100,
			UpdatedAt:           time.Now(),
			Tags:                []string{ /* I don't need any tags! */ },
			Blobs: []*godo.Blob{
				{
					Digest:              "sha256:123",
					CompressedSizeBytes: 123,
				},
				{
					Digest:              "sha256:456",
					CompressedSizeBytes: 456,
				},
			},
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
	testRepositoryV2 = do.RepositoryV2{
		RepositoryV2: &godo.RepositoryV2{
			RegistryName:   testRegistryName,
			Name:           testRegistryName,
			TagCount:       2,
			ManifestCount:  1,
			LatestManifest: testRepositoryManifest.RepositoryManifest,
		},
	}
	testRepositoryV2NoTags = do.RepositoryV2{
		RepositoryV2: &godo.RepositoryV2{
			RegistryName:   testRegistryName,
			Name:           testRegistryName,
			TagCount:       0,
			ManifestCount:  1,
			LatestManifest: testRepositoryManifestNoTags.RepositoryManifest,
		},
	}
	testDockerCredentials = &godo.DockerCredentials{
		// the base64 string is "username:password"
		DockerConfigJSON: []byte(`{"auths":{"hostname":{"auth":"dXNlcm5hbWU6cGFzc3dvcmQ="}}}`),
	}
	testGCBlobsDeleted    = uint64(42)
	testGCFreedBytes      = uint64(666)
	testGCStatus          = "requested"
	testGCUUID            = "gc-uuid"
	invalidGCUUID         = "invalid-gc-uuid"
	testTime              = time.Date(2020, 4, 1, 0, 0, 0, 0, time.UTC)
	testGarbageCollection = &do.GarbageCollection{
		&godo.GarbageCollection{
			UUID:         testGCUUID,
			RegistryName: testRegistryName,
			Status:       testGCStatus,
			CreatedAt:    testTime,
			UpdatedAt:    testTime,
			BlobsDeleted: testGCBlobsDeleted,
			FreedBytes:   testGCFreedBytes,
		},
	}
)

func TestRegistryCommand(t *testing.T) {
	cmd := Registry()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "get", "delete", "login", "logout", "options", "kubernetes-manifest", "repository", "docker-config", "garbage-collection")
}

func TestRepositoryCommand(t *testing.T) {
	cmd := Repository()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list", "list-v2", "list-manifests", "list-tags", "delete-manifest", "delete-tag")
}

func TestGarbageCollectionCommand(t *testing.T) {
	cmd := GarbageCollection()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get-active", "start", "cancel", "list")
}

func TestRegistryCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		rcr := godo.RegistryCreateRequest{
			Name:                 testRegistryName,
			SubscriptionTierSlug: testSubscriptionTier,
		}
		tm.registry.EXPECT().Create(&rcr).Return(&testRegistry, nil)
		config.Args = append(config.Args, testRegistryName)
		config.Doit.Set(config.NS, doctl.ArgSubscriptionTier, "basic")

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
		name          string
		readWrite     bool
		expirySeconds int
		expect        func(m *mocks.MockRegistryService)
	}{
		{
			name:          "read-only-no-expiry",
			readWrite:     false,
			expirySeconds: 0,
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
					ReadWrite: false,
				}).Return(testDockerCredentials, nil)
			},
		},
		{
			name:          "read-write-no-expiry",
			readWrite:     true,
			expirySeconds: 0,
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
					ReadWrite: true,
				}).Return(testDockerCredentials, nil)
			},
		},
		{
			name:          "read-only-with-expiry",
			readWrite:     false,
			expirySeconds: 3600,
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
					ReadWrite:     false,
					ExpirySeconds: godo.Int(3600),
				}).Return(testDockerCredentials, nil)
			},
		},
		{
			name:          "read-write-with-expiry",
			readWrite:     true,
			expirySeconds: 3600,
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
					ReadWrite:     true,
					ExpirySeconds: godo.Int(3600),
				}).Return(testDockerCredentials, nil)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if test.expect != nil {
					test.expect(tm.registry)
				}

				config.Doit.Set(config.NS, doctl.ArgReadWrite, test.readWrite)
				config.Doit.Set(config.NS, doctl.ArgRegistryExpirySeconds, test.expirySeconds)

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

func TestRepositoryListV2(t *testing.T) {
	t.Run("with latest tag", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.registry.EXPECT().Get().Return(&testRegistry, nil)
			tm.registry.EXPECT().ListRepositoriesV2(testRepositoryV2.RegistryName).Return([]do.RepositoryV2{testRepositoryV2}, nil)

			var buf bytes.Buffer
			config.Out = &buf
			err := RunListRepositoriesV2(config)
			assert.NoError(t, err)

			output := buf.String()
			// instead of trying to match the output, do some basic content checks
			assert.True(t, strings.Contains(output, testRepositoryV2.Name))
			assert.True(t, strings.Contains(output, testRepositoryV2.LatestManifest.Digest))
			// basic text format doesn't include blob data
			assert.False(t, strings.Contains(output, testRepositoryV2.LatestManifest.Blobs[0].Digest))
		})
	})
	t.Run("with <none> latest tag", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.registry.EXPECT().Get().Return(&testRegistry, nil)
			tm.registry.EXPECT().ListRepositoriesV2(testRepositoryV2NoTags.RegistryName).Return([]do.RepositoryV2{testRepositoryV2NoTags}, nil)

			var buf bytes.Buffer
			config.Out = &buf
			err := RunListRepositoriesV2(config)
			assert.NoError(t, err)

			output := buf.String()
			// instead of trying to match the output, do some basic content checks
			assert.True(t, strings.Contains(output, testRepositoryV2NoTags.Name))
			assert.True(t, strings.Contains(output, "<none>")) // default value when latest manifest has no tags
			// basic text format doesn't include blob data
			assert.False(t, strings.Contains(output, testRepositoryV2NoTags.LatestManifest.Blobs[0].Digest))
		})
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

func TestRepositoryListManifests(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Get().Return(&testRegistry, nil)
		tm.registry.EXPECT().ListRepositoryManifests(
			testRepositoryManifest.RegistryName,
			testRepositoryManifest.Repository,
		).Return([]do.RepositoryManifest{testRepositoryManifest}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = append(config.Args, testRepositoryManifest.Repository)

		err := RunListRepositoryManifests(config)
		assert.NoError(t, err)

		output := buf.String()
		// instead of trying to match the output, do some basic content checks
		assert.True(t, strings.Contains(output, testRepositoryManifest.Digest))
		assert.True(t, strings.Contains(output, fmt.Sprintf("%s", testRepositoryManifest.Tags)))
		// basic text format doesn't include blob data
		assert.False(t, strings.Contains(output, testRepositoryManifest.Blobs[0].Digest))
	})
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
			expectedAnnotations             map[string]string
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
			{
				argName:           "my-registry",
				argNamespace:      "kube-system",
				expectedName:      "my-registry",
				expectedNamespace: "kube-system",
				expectedAnnotations: map[string]string{
					DOSecretOperatorAnnotation: "my-registry",
				},
			},
		}

		// mocks shared across both test cases
		// Get should be called only when a name isn't supplied, to look up the registry's name
		tm.registry.EXPECT().Get().Return(&testRegistry, nil).Times(1)
		// DockerCredentials should be called both times to retrieve the credentials
		tm.registry.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
			ReadWrite: false,
		}).Return(testDockerCredentials, nil).Times(len(tcs))

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
			assert.Equal(t, tc.expectedAnnotations, secret.Annotations)
		}
	})
}

func TestRegistryLogin(t *testing.T) {
	tests := []struct {
		name          string
		expirySeconds int
		expect        func(m *mocks.MockRegistryService)
	}{
		{
			name:          "no-expiry",
			expirySeconds: 0,
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Endpoint().Return(do.RegistryHostname)
				m.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
					ReadWrite: true,
				}).Return(testDockerCredentials, nil)
			},
		},
		{
			name:          "with-expiry",
			expirySeconds: 3600,
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Endpoint().Return(do.RegistryHostname)
				m.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
					ReadWrite:     true,
					ExpirySeconds: godo.Int(3600),
				}).Return(testDockerCredentials, nil)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if test.expect != nil {
					test.expect(tm.registry)
				}

				config.Doit.Set(config.NS, doctl.ArgRegistryExpirySeconds, test.expirySeconds)

				config.Out = os.Stderr
				err := RunRegistryLogin(config)
				assert.NoError(t, err)
			})
		})
	}
}

func TestRegistryLogout(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgRegistryAuthorizationServerEndpoint, "http://example.com")
		tm.registry.EXPECT().Endpoint().Return(do.RegistryHostname)
		tm.registry.EXPECT().RevokeOAuthToken(gomock.Any(), "http://example.com").Times(1).Return(nil)

		config.Out = os.Stderr
		err := RunRegistryLogout(config)
		assert.NoError(t, err)
	})
}

func TestGarbageCollectionStart(t *testing.T) {
	defaultStartGCRequest := &godo.StartGarbageCollectionRequest{
		Type: godo.GCTypeUnreferencedBlobsOnly,
	}
	tests := []struct {
		name      string
		extraArgs []string

		expect      func(m *mocks.MockRegistryService, config *CmdConfig)
		expectError error
	}{
		{
			name: "without registry name arg",
		},
		{
			name: "with registry name arg",
			extraArgs: []string{
				testRegistryName,
			},
			expect: func(m *mocks.MockRegistryService, config *CmdConfig) {
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				m.EXPECT().StartGarbageCollection(testRegistry.Name, defaultStartGCRequest).Return(testGarbageCollection, nil)
			},
		},
		{
			name: "include untagged manifests",
			extraArgs: []string{
				testRegistryName,
			},
			expect: func(m *mocks.MockRegistryService, config *CmdConfig) {
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				config.Doit.Set(config.NS, doctl.ArgGCIncludeUntaggedManifests, true)
				config.Doit.Set(config.NS, doctl.ArgGCExcludeUnreferencedBlobs, false)
				m.EXPECT().StartGarbageCollection(testRegistry.Name, &godo.StartGarbageCollectionRequest{
					Type: godo.GCTypeUntaggedManifestsAndUnreferencedBlobs,
				}).Return(testGarbageCollection, nil)
			},
		},
		{
			name: "include untagged manifests, exclude unreferenced blobs",
			extraArgs: []string{
				testRegistryName,
			},
			expect: func(m *mocks.MockRegistryService, config *CmdConfig) {
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				config.Doit.Set(config.NS, doctl.ArgGCIncludeUntaggedManifests, true)
				config.Doit.Set(config.NS, doctl.ArgGCExcludeUnreferencedBlobs, true)
				m.EXPECT().StartGarbageCollection(testRegistry.Name, &godo.StartGarbageCollectionRequest{
					Type: godo.GCTypeUntaggedManifestsOnly,
				}).Return(testGarbageCollection, nil)
			},
		},
		{
			name: "fail with invalid combination of gc targets",
			extraArgs: []string{
				invalidRegistryName,
			},
			expect: func(m *mocks.MockRegistryService, config *CmdConfig) {
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				config.Doit.Set(config.NS, doctl.ArgGCIncludeUntaggedManifests, false)
				config.Doit.Set(config.NS, doctl.ArgGCExcludeUnreferencedBlobs, true)
			},
			expectError: fmt.Errorf("incompatible combination of include-untagged-manifests and exclude-unreferenced-blobs flags"),
		},
		{
			name: "fail with invalid registry name arg",
			extraArgs: []string{
				invalidRegistryName,
			},
			expect: func(m *mocks.MockRegistryService, config *CmdConfig) {
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				m.EXPECT().StartGarbageCollection(invalidRegistryName, defaultStartGCRequest).Return(nil, fmt.Errorf("meow"))
			},
			expectError: fmt.Errorf("meow"),
		},
		{
			name: "fail with too many args",
			extraArgs: []string{
				invalidRegistryName,
				testGCUUID,
			},
			expect:      func(m *mocks.MockRegistryService, config *CmdConfig) {},
			expectError: fmt.Errorf("(test) command contains unsupported arguments"),
		},
		{
			name: "prompt to confirm without --force argument",
			extraArgs: []string{
				testRegistryName,
			},
			expect:      func(m *mocks.MockRegistryService, config *CmdConfig) {},
			expectError: errOperationAborted,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if test.expect != nil {
					test.expect(tm.registry, config)
				} else {
					config.Doit.Set(config.NS, doctl.ArgForce, true)
					tm.registry.EXPECT().Get().Return(&testRegistry, nil)
					tm.registry.EXPECT().StartGarbageCollection(testRegistry.Name, defaultStartGCRequest).Return(testGarbageCollection, nil)
				}

				if test.extraArgs != nil {
					config.Args = append(config.Args, test.extraArgs...)
				}

				err := RunStartGarbageCollection(config)

				if test.expectError != nil {
					assert.Error(t, err)
					assert.Equal(t, test.expectError.Error(), err.Error())
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestGarbageCollectionGetActive(t *testing.T) {
	tests := []struct {
		name        string
		extraArgs   []string
		expect      func(m *mocks.MockRegistryService)
		expectError error
	}{
		{
			name: "without registry name arg",
		},
		{
			name: "with registry name arg",
			extraArgs: []string{
				testRegistryName,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().GetGarbageCollection(testRegistry.Name).Return(testGarbageCollection, nil)
			},
		},
		{
			name: "fail with invalid registry name arg",
			extraArgs: []string{
				invalidRegistryName,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().GetGarbageCollection(invalidRegistryName).Return(nil, fmt.Errorf("meow"))
			},
			expectError: fmt.Errorf("meow"),
		},
		{
			name: "fail with too many args",
			extraArgs: []string{
				invalidRegistryName,
				testRegistryName,
			},
			expect:      func(m *mocks.MockRegistryService) {},
			expectError: fmt.Errorf("(test) command contains unsupported arguments"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if test.expect != nil {
					test.expect(tm.registry)
				} else {
					tm.registry.EXPECT().Get().Return(&testRegistry, nil)
					tm.registry.EXPECT().GetGarbageCollection(testRegistry.Name).Return(testGarbageCollection, nil)
				}

				if test.extraArgs != nil {
					config.Args = append(config.Args, test.extraArgs...)
				}

				err := RunGetGarbageCollection(config)

				if test.expectError != nil {
					assert.Error(t, err)
					assert.Equal(t, test.expectError.Error(), err.Error())
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestGarbageCollectionList(t *testing.T) {
	tests := []struct {
		name      string
		extraArgs []string

		expect      func(m *mocks.MockRegistryService)
		expectError error
	}{
		{
			name: "without registry name arg",
		},
		{
			name: "with registry name arg",
			extraArgs: []string{
				testRegistryName,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().ListGarbageCollections(testRegistry.Name).Return([]do.GarbageCollection{*testGarbageCollection}, nil)
			},
		},
		{
			name: "fail with invalid registry name arg",
			extraArgs: []string{
				invalidRegistryName,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().ListGarbageCollections(invalidRegistryName).Return(nil, fmt.Errorf("meow"))
			},
			expectError: fmt.Errorf("meow"),
		},
		{
			name: "fail with too many args",
			extraArgs: []string{
				invalidRegistryName,
				testGCUUID,
			},
			expect:      func(m *mocks.MockRegistryService) {},
			expectError: fmt.Errorf("(test) command contains unsupported arguments"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if test.expect != nil {
					test.expect(tm.registry)
				} else {
					tm.registry.EXPECT().Get().Return(&testRegistry, nil)
					tm.registry.EXPECT().ListGarbageCollections(testRegistry.Name).Return([]do.GarbageCollection{*testGarbageCollection}, nil)
				}

				if test.extraArgs != nil {
					config.Args = append(config.Args, test.extraArgs...)
				}

				err := RunListGarbageCollections(config)

				if test.expectError != nil {
					assert.Error(t, err)
					assert.Equal(t, test.expectError.Error(), err.Error())
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestGarbageCollectionUpdate(t *testing.T) {
	tests := []struct {
		name        string
		extraArgs   []string
		expect      func(m *mocks.MockRegistryService)
		expectError error
	}{
		{
			name: "with gc uuid arg",
			extraArgs: []string{
				testGCUUID,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Get().Return(&testRegistry, nil)
				m.EXPECT().CancelGarbageCollection(testRegistry.Name, testGCUUID).Return(testGarbageCollection, nil)
			},
		},
		{
			name: "with registry name and gc uuid arg",
			extraArgs: []string{
				testRegistryName,
				testGCUUID,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().CancelGarbageCollection(testRegistryName, testGCUUID).Return(testGarbageCollection, nil)
			},
		},
		{
			name: "fail with invalid registry name arg",
			extraArgs: []string{
				invalidRegistryName,
				testGCUUID,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().CancelGarbageCollection(invalidRegistryName, testGCUUID).Return(nil, fmt.Errorf("meow"))
			},
			expectError: fmt.Errorf("meow"),
		},
		{
			name: "fail with invalid gc uuid arg and valid registry name arg",
			extraArgs: []string{
				testRegistryName,
				invalidGCUUID,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().CancelGarbageCollection(testRegistryName, invalidGCUUID).Return(nil, fmt.Errorf("meow"))
			},
			expectError: fmt.Errorf("meow"),
		},
		{
			name: "fail with invalid gc uuid arg",
			extraArgs: []string{
				invalidGCUUID,
			},
			expect: func(m *mocks.MockRegistryService) {
				m.EXPECT().Get().Return(&testRegistry, nil)
				m.EXPECT().CancelGarbageCollection(testRegistryName, invalidGCUUID).Return(nil, fmt.Errorf("meow"))
			},
			expectError: fmt.Errorf("meow"),
		},
		{
			name: "fail with too many args",
			extraArgs: []string{
				invalidRegistryName,
				testGCUUID,
				testGCUUID,
			},
			expect:      func(m *mocks.MockRegistryService) {},
			expectError: fmt.Errorf("(test) command contains unsupported arguments"),
		},
		{
			name:        "fail with no args",
			extraArgs:   []string{},
			expect:      func(m *mocks.MockRegistryService) {},
			expectError: fmt.Errorf("(test) command is missing required arguments"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				test.expect(tm.registry)

				if test.extraArgs != nil {
					config.Args = append(config.Args, test.extraArgs...)
				}

				err := RunCancelGarbageCollection(config)

				if test.expectError != nil {
					assert.Error(t, err)
					assert.Equal(t, test.expectError.Error(), err.Error())
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}
