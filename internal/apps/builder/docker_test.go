package builder

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/digitalocean/doctl/commands/charm/text"
	"github.com/digitalocean/godo"
	"github.com/docker/docker/api/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerComponentBuild_validation(t *testing.T) {
	ctx := context.Background()

	t.Run("static site - missing output dir", func(t *testing.T) {
		spec := &godo.AppSpec{
			StaticSites: []*godo.AppStaticSiteSpec{{
				DockerfilePath: "./Dockerfile",
				SourceDir:      "./subdir",
				Name:           "web",
			}},
		}
		builder := &DockerComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				spec:       spec,
				component:  spec.StaticSites[0],
				contextDir: t.TempDir(),
			},
			dockerComponent: spec.StaticSites[0],
		}
		_, err := builder.Build(ctx)
		require.EqualError(t, err, "output_dir is required for dockerfile-based static site builds")
	})

	t.Run("static site - output dir not absolute", func(t *testing.T) {
		spec := &godo.AppSpec{
			StaticSites: []*godo.AppStaticSiteSpec{{
				Name:           "web",
				DockerfilePath: "./Dockerfile",
				OutputDir:      "test",
			}},
		}
		builder := &DockerComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				spec:       spec,
				component:  spec.StaticSites[0],
				contextDir: t.TempDir(),
			},
			dockerComponent: spec.StaticSites[0],
		}
		_, err := builder.Build(ctx)
		require.EqualError(t, err, "output_dir must be an absolute path with dockerfile-based static site builds")
	})

	t.Run("static site - output dir is /", func(t *testing.T) {
		spec := &godo.AppSpec{
			StaticSites: []*godo.AppStaticSiteSpec{{
				Name:           "web",
				DockerfilePath: "./Dockerfile",
				OutputDir:      "/",
			}},
		}
		builder := &DockerComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				spec:       spec,
				component:  spec.StaticSites[0],
				contextDir: t.TempDir(),
			},
			dockerComponent: spec.StaticSites[0],
		}
		_, err := builder.Build(ctx)
		require.EqualError(t, err, "output_dir may not be /")
	})
}

func TestDockerComponentBuild(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	t.Run("no component", func(t *testing.T) {
		builder := &DockerComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				contextDir: t.TempDir(),
			},
		}
		_, err := builder.Build(ctx)
		require.ErrorContains(t, err, "no component")
	})

	t.Run("happy path - service", func(t *testing.T) {
		service := &godo.AppServiceSpec{
			DockerfilePath: "./Dockerfile",
			Name:           "web",
			Envs: []*godo.AppVariableDefinition{
				{
					Key:   "build-arg-1",
					Value: "build-val-1",
					Type:  godo.AppVariableType_General,
					Scope: godo.AppVariableScope_BuildTime,
				},
				{
					Key:   "override-1",
					Value: "newval",
				},
				{
					Key:   "run-build-arg-1",
					Value: "run-build-val-1",
					Type:  godo.AppVariableType_General,
					Scope: godo.AppVariableScope_RunAndBuildTime,
				},
				{
					Key:   "run-arg-1",
					Value: "run-val-1",
					Type:  godo.AppVariableType_General,
					Scope: godo.AppVariableScope_RunTime,
				},
				{
					Key:   "secret-arg-1",
					Value: "secret-val-1",
					Type:  godo.AppVariableType_Secret,
				},
			},
		}
		spec := &godo.AppSpec{
			Services: []*godo.AppServiceSpec{service},
			Envs: []*godo.AppVariableDefinition{
				{
					Key:   "override-1",
					Value: "override-1",
				},
			},
		}

		mockClient := NewMockDockerEngineClient(ctrl)
		logBuf := newLoggableBuffer(t)
		builder := &DockerComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli:                  mockClient,
				spec:                 spec,
				component:            service,
				buildCommandOverride: "test",
				logWriter:            &logBuf,
				noCache:              true,
				contextDir:           t.TempDir(),
			},
			dockerComponent: service,
		}

		mockClient.EXPECT().ImageBuild(ctx, gomock.Any(), types.ImageBuildOptions{
			Dockerfile: "Dockerfile",
			Tags: []string{
				builder.AppImageOutputName(),
			},
			BuildArgs: map[string]*string{
				"build-arg-1":     strPtr("build-val-1"),
				"override-1":      strPtr("newval"),
				"run-build-arg-1": strPtr("run-build-val-1"),
			},
			NoCache: true,
		}).Return(types.ImageBuildResponse{
			Body: ioutil.NopCloser(strings.NewReader("")),
		}, nil)

		_, err := builder.Build(ctx)
		require.NoError(t, err)

		assert.Contains(t, logBuf.String(), text.Crossmark.String()+" build command overrides are ignored for dockerfile-based builds")
	})

	t.Run("happy path - static site", func(t *testing.T) {
		site := &godo.AppStaticSiteSpec{
			DockerfilePath: "Dockerfile",
			SourceDir:      "subdir",
			Name:           "web",
			OutputDir:      "/app/public",
			Envs: []*godo.AppVariableDefinition{
				{
					Key:   "build-arg-1",
					Value: "build-val-1",
					Type:  godo.AppVariableType_General,
					Scope: godo.AppVariableScope_BuildTime,
				},
			},
		}
		spec := &godo.AppSpec{
			StaticSites: []*godo.AppStaticSiteSpec{site},
			Envs: []*godo.AppVariableDefinition{
				{
					Key:   "override-1",
					Value: "override-1",
				},
			},
		}

		mockClient := NewMockDockerEngineClient(ctrl)
		logBuf := newLoggableBuffer(t)
		contextDir := t.TempDir()
		err := os.Mkdir(filepath.Join(contextDir, "subdir"), 0775)
		require.NoError(t, err)
		// Dockerfile is outside of source_dir
		err = ioutil.WriteFile(filepath.Join(contextDir, "Dockerfile"), []byte("FROM scratch"), 0664)
		require.NoError(t, err)

		builder := &DockerComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli:        mockClient,
				spec:       spec,
				component:  site,
				logWriter:  &logBuf,
				contextDir: contextDir,
			},
			dockerComponent: site,
		}

		gomock.InOrder(
			// app image build
			mockClient.EXPECT().ImageBuild(ctx, gomock.Any(), &delegatedMatcher{
				Description: "matches app container image build options",
				MatchesFunc: func(x any) bool {
					options, ok := x.(types.ImageBuildOptions)
					if !ok {
						return false
					}
					t := &testingT{}
					assert.Equal(t, []string{builder.AppImageOutputName()}, options.Tags)
					assert.Equal(t, map[string]*string{
						"build-arg-1": strPtr("build-val-1"),
						"override-1":  strPtr("override-1"),
					}, options.BuildArgs)
					return !t.Failed()
				},
			}).
				DoAndReturn(func(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
					// options.Dockerfile is added to the archive as a fake .dockerfile.{random} path as it's outside the source dir
					assert.Regexp(t, regexp.MustCompile(`^\.dockerfile\.[a-z0-9]+$`), options.Dockerfile)
					return types.ImageBuildResponse{
						Body: io.NopCloser(strings.NewReader("")),
					}, nil
				}),

			// static site build
			mockClient.EXPECT().
				ImageBuild(ctx, gomock.Any(),
					&delegatedMatcher{
						Description: "matches static site image build options",
						MatchesFunc: func(x any) bool {
							options, ok := x.(types.ImageBuildOptions)
							if !ok {
								return false
							}

							t := &testingT{}
							assert.Equal(t, "./Dockerfile.static", options.Dockerfile)
							assert.Equal(t, []string{builder.StaticSiteImageOutputName()}, options.Tags)
							assert.Equal(t, map[string]*string{
								"app_image":   strPtr(builder.AppImageOutputName()),
								"nginx_image": strPtr(StaticSiteNginxImage),
								"output_dir":  strPtr(site.GetOutputDir()),
							}, options.BuildArgs)
							return !t.Failed()
						},
					}).
				DoAndReturn(func(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
					// assert the contents of the archive
					t.Run("static site archive", func(t *testing.T) {
						assertArchiveContents(t, context, map[string]func(t *testing.T, h *tar.Header, c []byte){
							"Dockerfile.static": func(t *testing.T, h *tar.Header, c []byte) {
								assert.Equal(t, `
ARG app_image
ARG nginx_image
ARG output_dir
FROM ${app_image} as content
FROM ${nginx_image}
ARG output_dir

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=content ${output_dir}/ /www
`,
									string(c))
							},
							"nginx.conf": func(t *testing.T, h *tar.Header, c []byte) {
								assert.Equal(t, `
server {
	listen 8080;
	listen [::]:8080;

	resolver 127.0.0.11;
	autoindex off;

	server_name _;
	server_tokens off;

	root /www;
	gzip_static on;
}
`,
									string(c))
							},
						})
					})

					return types.ImageBuildResponse{
						Body: io.NopCloser(strings.NewReader("")),
					}, nil
				}),
		)

		_, err = builder.Build(ctx)
		require.NoError(t, err)
	})
}

type delegatedMatcher struct {
	Description string
	MatchesFunc func(x any) bool
}

func (m *delegatedMatcher) Matches(x any) bool {
	return m.MatchesFunc(x)
}

func (m *delegatedMatcher) String() string {
	return m.Description
}

// testingT allows the use of stretchr/assert methods to produce a boolean value
type testingT struct {
	failed bool
	mtx    sync.Mutex
	errors []string
}

func (t *testingT) Errorf(format string, args ...interface{}) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.failed = true
	t.errors = append(t.errors, fmt.Sprintf(format, args...))
}

func (t *testingT) FailNow() {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.failed = true
}

func (t *testingT) Failed() bool {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.failed
}

func (t *testingT) Error() string {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return strings.Join(t.errors, "\n")
}

func newLoggableBuffer(t *testing.T) bytes.Buffer {
	var buf bytes.Buffer
	t.Cleanup(func() {
		if t.Failed() {
			t.Log(buf.String())
		}
	})
	return buf
}

func assertArchiveContents(t *testing.T, archive io.Reader, tcs map[string]func(t *testing.T, header *tar.Header, content []byte)) {
	r := tar.NewReader(archive)
	filesFound := make(map[string]struct{})
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		name := header.Name
		tc := tcs[name]
		if tc == nil {
			t.Fatalf("unexpected file %s", name)
		}

		content, err := ioutil.ReadAll(r)
		require.NoError(t, err, "reading %s", name)

		filesFound[name] = struct{}{}
		tc(t, header, content)
	}

	for name := range tcs {
		if _, ok := filesFound[name]; !ok {
			t.Fatalf("missing expected file %s", name)
		}
	}
}
