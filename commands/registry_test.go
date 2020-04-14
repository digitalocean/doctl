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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
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
	assertCommandNames(t, cmd, "create", "get", "delete", "login", "logout", "kubernetes-manifest", "repository")
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
	extraTag := "extra-tag"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Get().Return(&testRegistry, nil)
		tm.registry.EXPECT().DeleteTag(
			testRepositoryTag.RegistryName,
			testRepositoryTag.Repository,
			testRepositoryTag.Tag,
		).Return(nil)
		tm.registry.EXPECT().DeleteTag(
			testRepositoryTag.RegistryName,
			testRepositoryTag.Repository,
			extraTag,
		).Return(nil)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		config.Args = append(config.Args, testRepositoryTag.Repository, testRepositoryTag.Tag, extraTag)

		err := RunRepositoryDeleteTag(config)
		assert.NoError(t, err)
	})
}

func TestRepositoryDeleteManifest(t *testing.T) {
	extraDigest := "sha256:5b0bcabd1ed22e9fb1310cf6c2dec7cdef19f0ad69efa1f392e94a4333501270"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Get().Return(&testRegistry, nil)
		tm.registry.EXPECT().DeleteManifest(
			testRepositoryTag.RegistryName,
			testRepositoryTag.Repository,
			testRepositoryTag.ManifestDigest,
		).Return(nil)
		tm.registry.EXPECT().DeleteManifest(
			testRepositoryTag.RegistryName,
			testRepositoryTag.Repository,
			extraDigest,
		).Return(nil)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		config.Args = append(config.Args, testRepositoryTag.Repository, testRepositoryTag.ManifestDigest, extraDigest)

		err := RunRepositoryDeleteManifest(config)
		assert.NoError(t, err)
	})
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

// https://npf.io/2015/06/testing-exec-command/
func fakeExecCommand(testName string) func(string, ...string) *exec.Cmd {
	return func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.v", "-test.run=TestHelperProcess", "--", testName, command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	}
}

// this function adds a fake "docker" to PATH because the commands look
// for it but the tests fake the call through fakeExecCommand
// so the cli binary isn't needed at all
func createFakeDocker() (func(), error) {
	// get the original PATH and create a temp dir
	originalPath := os.Getenv("PATH")
	tempDir, err := ioutil.TempDir("", "docker")
	if err != nil {
		return func() {}, err
	}

	// define this here so we can return it
	// in different places
	fnCleanUp := func() {
		os.Setenv("PATH", originalPath)
		os.RemoveAll(tempDir)
	}

	// create a fake executable
	f, err := os.Create(filepath.Join(tempDir, "docker"))
	if err != nil {
		return fnCleanUp, err
	}
	defer f.Close()
	f.Chmod(0777)

	// add the temp dir to PATH
	newPath := fmt.Sprintf("%s%c%s", tempDir, os.PathListSeparator, originalPath)
	err = os.Setenv("PATH", newPath)

	// return a function to clean up
	return fnCleanUp, err
}

func TestRegistryLogin(t *testing.T) {
	// create a fake docker executable in PATH so the test doesn't fail
	// if Docker isn't installed
	removeFakeDocker, err := createFakeDocker()
	defer removeFakeDocker() // clean up even if there is an error
	if err != nil {
		t.Logf("couldn't create temp dir to hold the fake docker cli binary: %v", err)
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Endpoint().Return(do.RegistryHostname)
		tm.registry.EXPECT().DockerCredentials(&godo.RegistryDockerCredentialsRequest{
			ReadWrite: true,
		}).Return(testDockerCredentials, nil)

		// fake execCommand
		execCommand = fakeExecCommand("login")
		defer func() { execCommand = exec.Command }()

		config.Out = os.Stderr
		err := RunRegistryLogin(config)
		assert.NoError(t, err)
	})
}

func TestRegistryLogout(t *testing.T) {
	// create a fake docker executable in PATH so the test doesn't fail
	// if Docker isn't installed
	removeFakeDocker, err := createFakeDocker()
	defer removeFakeDocker() // clean up even if there is an error
	if err != nil {
		t.Logf("couldn't create temp dir to hold the fake docker cli binary: %v", err)
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Endpoint().Return(do.RegistryHostname)

		// fake execCommand
		execCommand = fakeExecCommand("logout")
		defer func() { execCommand = exec.Command }()

		config.Out = os.Stderr
		err := RunRegistryLogout(config)
		assert.NoError(t, err)
	})
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		t.Log("Called TestHelperProcess() when it's not needed")
		return
	}

	// This process is used to fake the "docker" cli command call.
	if len(os.Args) < 4 {
		t.Error("Invalid TestHelperProcess() call. Use fakeExecCommand() to generate the command")
		return
	}

	switch os.Args[4] {
	case "login":
		// Make sure we receive the correct test credentials
		// as in `testDockerCredentials`.
		expectedCommand := []string{"docker", "login", "hostname", "-u", "username", "--password-stdin"}
		gotCommand := os.Args[5:]

		assert.Equal(t, expectedCommand, gotCommand)

		stdin, err := ioutil.ReadAll(os.Stdin)
		assert.NoError(t, err)

		gotPassword := string(stdin)
		assert.Equal(t, "password", gotPassword)

	case "logout":
		expected := []string{"docker", "logout", do.RegistryHostname}
		got := os.Args[5:]

		assert.Equal(t, expected, got)

	default:
		t.Error("Unknown test", os.Args[4])
	}
}
