package apps

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAppSpec(t *testing.T) {
	expectedSpec := validAppSpec

	t.Run("json", func(t *testing.T) {
		spec, err := ParseAppSpec([]byte(validJSONSpec))
		require.NoError(t, err)
		assert.Equal(t, expectedSpec, spec)
	})
	t.Run("yaml", func(t *testing.T) {
		spec, err := ParseAppSpec([]byte(validYAMLSpec))
		require.NoError(t, err)
		assert.Equal(t, expectedSpec, spec)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := ParseAppSpec([]byte("invalid spec"))
		require.Error(t, err)
	})
	t.Run("unknown fields", func(t *testing.T) {
		_, err := ParseAppSpec([]byte(unknownFieldSpec))
		require.Error(t, err)
	})
}

func Test_readAppSpec(t *testing.T) {
	tcs := []struct {
		name  string
		setup func(t *testing.T) (path string, stdin io.Reader)

		wantSpec *godo.AppSpec
		wantErr  error
	}{
		{
			name: "stdin",
			setup: func(t *testing.T) (string, io.Reader) {
				return "-", bytes.NewBufferString(validYAMLSpec)
			},
			wantSpec: validAppSpec,
		},
		{
			name: "file yaml",
			setup: func(t *testing.T) (string, io.Reader) {
				return testTempFile(t, []byte(validJSONSpec)), nil
			},
			wantSpec: validAppSpec,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			path, stdin := tc.setup(t)
			spec, err := ReadAppSpec(stdin, path)
			if tc.wantErr != nil {
				require.Equal(t, tc.wantErr, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantSpec, spec)
		})
	}
}

const (
	validJSONSpec = `{
	"name": "test",
	"services": [
		{
			"name": "web",
			"github": {
				"repo": "digitalocean/sample-golang",
				"branch": "main"
			}
		}
	],
	"static_sites": [
		{
			"name": "static",
			"git": {
				"repo_clone_url": "git@github.com:digitalocean/sample-gatsby.git",
				"branch": "main"
			},
			"routes": [
				{
				"path": "/static"
				}
			]
		}
	]
}`
	validYAMLSpec = `name: test
services:
- github:
    branch: main
    repo: digitalocean/sample-golang
  name: web
static_sites:
- git:
    branch: main
    repo_clone_url: git@github.com:digitalocean/sample-gatsby.git
  name: static
  routes:
  - path: /static
`
	unknownFieldSpec = `
name: test
bugField: bad
services:
- name: web
  github:
    repo: digitalocean/sample-golang
    branch: main
static_sites:
- name: static
  git:
    repo_clone_url: git@github.com:digitalocean/sample-gatsby.git
    branch: main
  routes:
  - path: /static
`
)

var validAppSpec = &godo.AppSpec{
	Name: "test",
	Services: []*godo.AppServiceSpec{
		{
			Name: "web",
			GitHub: &godo.GitHubSourceSpec{
				Repo:   "digitalocean/sample-golang",
				Branch: "main",
			},
		},
	},
	StaticSites: []*godo.AppStaticSiteSpec{
		{
			Name: "static",
			Git: &godo.GitSourceSpec{
				RepoCloneURL: "git@github.com:digitalocean/sample-gatsby.git",
				Branch:       "main",
			},
			Routes: []*godo.AppRouteSpec{
				{Path: "/static"},
			},
		},
	},
}

func testTempFile(t *testing.T, data []byte) string {
	t.Helper()
	file := t.TempDir() + "/file"
	err := ioutil.WriteFile(file, data, 0644)
	require.NoError(t, err, "writing temp file")
	return file
}
