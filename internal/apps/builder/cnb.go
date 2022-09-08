package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/godo"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/archive"
	"github.com/kballard/go-shellquote"
)

const (
	// CNBBuilderImage represents the local cnb builder.
	CNBBuilderImage = "digitaloceanapps/cnb-local-builder:v0.49.0"

	appVarAllowListKey = "APP_VARS"
	appVarPrefix       = "APP_VAR_"
	cnbCacheDir        = "/cnb/cache"
)

// CNBComponentBuilder represents a CNB builder.
type CNBComponentBuilder struct {
	baseComponentBuilder
	versioning    CNBVersioning
	localCacheDir string
}

// CNBVersioning contains CNB versioning config.
type CNBVersioning struct {
	Buildpacks []*Buildpack
}

// Buildpack represents a CNB buildpack.
type Buildpack struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
}

// Build attempts to build the requested component using the CNB Builder.
func (b *CNBComponentBuilder) Build(ctx context.Context) (res ComponentBuilderResult, err error) {
	if b.component == nil {
		return res, errors.New("no component was provided for the build")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return res, err
	}

	env, err := b.cnbEnv(ctx)
	if err != nil {
		return res, fmt.Errorf("configuring environment variables: %w", err)
	}

	sourceDockerSock, err := filepath.EvalSymlinks("/var/run/docker.sock")
	if err != nil {
		return res, err
	}

	mounts := []mount.Mount{{
		Type:   mount.TypeBind,
		Source: sourceDockerSock,
		Target: "/var/run/docker.sock",
	}}
	if !b.copyOnWriteSemantics {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: cwd,
			Target: "/workspace",
		})
	}

	if b.localCacheDir != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: b.localCacheDir,
			Target: cnbCacheDir,
		})
	}

	buildContainer, err := b.cli.ContainerCreate(ctx, &container.Config{
		Image:        CNBBuilderImage,
		Entrypoint:   []string{"sh", "-c", "sleep infinity"},
		AttachStdout: true,
		AttachStderr: true,
	}, &container.HostConfig{
		Mounts: mounts,
	}, nil, nil, "")
	if err != nil {
		return res, err
	}

	start := time.Now()
	defer func() {
		res.BuildDuration = time.Since(start)
		// we use context.Background() so we can remove the container if the original context is cancelled.
		ctx := context.Background()

		err = b.cli.ContainerRemove(ctx, buildContainer.ID, types.ContainerRemoveOptions{
			Force: true,
		})
	}()

	if err := b.cli.ContainerStart(ctx, buildContainer.ID, types.ContainerStartOptions{}); err != nil {
		return res, err
	}

	if b.copyOnWriteSemantics {
		// Prepare source copy info.
		srcInfo, err := archive.CopyInfoSourcePath(b.contextDir, true)
		if err != nil {
			return res, err
		}
		srcArchive, err := archive.TarResource(srcInfo)
		if err != nil {
			return res, fmt.Errorf("preparing build context: %w", err)
		}
		defer srcArchive.Close()
		dstInfo := archive.CopyInfo{
			Path:  "/workspace",
			IsDir: true,
		}
		archDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
		if err != nil {
			return res, err
		}
		defer preparedArchive.Close()
		err = b.cli.CopyToContainer(ctx, buildContainer.ID, archDir, preparedArchive, types.CopyToContainerOptions{
			AllowOverwriteDirWithFile: false,
			CopyUIDGID:                false,
		})
		if err != nil {
			return res, err
		}
	}

	err = b.runExec(
		ctx,
		buildContainer.ID,
		[]string{"sh", "-c", "/.app_platform/build.sh"},
		env,
		b.getLogWriter(),
	)
	if err != nil {
		return res, err
	}

	if b.component.GetType() == godo.AppComponentTypeStaticSite {
		var workspacePathB bytes.Buffer
		err = b.runExec(
			ctx,
			buildContainer.ID,
			[]string{"sh", "-c", "cat /.app_platform/local/WORKSPACE_PATH"},
			nil,
			&workspacePathB,
		)
		if err != nil {
			return res, err
		}
		workspacePath := workspacePathB.String()
		template.Render(b.getLogWriter(), heredoc.Doc(`
			{{success checkmark}} workspace path
			{{highlight .}}
		`,
		), charm.IndentString(4, workspacePath))

		var assetsPathB bytes.Buffer
		err = b.runExec(
			ctx,
			buildContainer.ID,
			[]string{"sh", "-c", "cat /.app_platform/local/ASSETS_PATH"},
			nil,
			&assetsPathB,
		)
		if err != nil {
			return res, err
		}
		assetsPath := assetsPathB.String()
		template.Render(b.getLogWriter(), heredoc.Doc(`
			{{success checkmark}} assets path
			{{highlight .}}
		`,
		), charm.IndentString(4, assetsPath))

		err = b.runExec(
			ctx,
			buildContainer.ID,
			[]string{"sh", "-c", `cat << EOF > "/workspace/nginx.conf"
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
EOF`,
			},
			nil,
			ioutil.Discard,
		)

		var nginxConf bytes.Buffer
		err = b.runExec(
			ctx,
			buildContainer.ID,
			[]string{"sh", "-c", "cat /workspace/nginx.conf"},
			nil,
			&nginxConf,
		)
		if err != nil {
			return res, err
		}
		template.Render(b.getLogWriter(), heredoc.Doc(`
			{{success checkmark}} nginxConf
			{{highlight .}}
		`,
		), charm.IndentString(4, nginxConf.String()))

		assetsPath = strings.TrimPrefix(assetsPath, workspacePath)
		if assetsPath == "" {
			assetsPath = "."
		} else {
			assetsPath = "./" + assetsPath
		}
		err = b.runExec(
			ctx,
			buildContainer.ID,
			[]string{"sh", "-c", fmt.Sprintf(`cat << EOF > "/workspace/Dockerfile.static"
FROM nginx:alpine

COPY %s /www
RUN rm -f /www/nginx.conf

COPY ./nginx.conf /etc/nginx/conf.d/default.conf
EOF`, assetsPath),
			},
			nil,
			ioutil.Discard,
		)
		if err != nil {
			return res, err
		}

		var dockerStatic bytes.Buffer
		err = b.runExec(
			ctx,
			buildContainer.ID,
			[]string{"sh", "-c", "cat /workspace/Dockerfile.static"},
			nil,
			&nginxConf,
		)
		if err != nil {
			return res, err
		}
		template.Render(b.getLogWriter(), heredoc.Doc(`
			{{success checkmark}} dockerSTatic
			{{highlight .}}
		`,
		), charm.IndentString(4, dockerStatic.String()))

		err = b.runExec(
			ctx,
			buildContainer.ID,
			[]string{
				"sh", "-c",
				shellquote.Join(
					"docker", "build",
					"-t", b.StaticSiteImageOutputName(),
					"-f", workspacePath+"/Dockerfile.static",
					workspacePath,
				),
			},
			nil,
			b.getLogWriter(),
		)
		if err != nil {
			return res, err
		}
	}

	return res, nil
}

func (b *CNBComponentBuilder) cnbEnv(ctx context.Context) ([]string, error) {
	envs := []string{}
	appVars := []string{}

	envMap, err := b.getEnvMap()
	if err != nil {
		return nil, err
	}
	for k, v := range envMap {
		envs = append(envs, appVarPrefix+k+"="+v)
		appVars = append(appVars, k)
	}
	if len(appVars) > 0 {
		sort.Strings(appVars)
		envs = append(envs, appVarAllowListKey+"="+strings.Join(appVars, ","))
	}

	envs = append(envs, "CNB_UPLOAD_RETRY=1")
	envs = append(envs, "APP_IMAGE_URL="+b.AppImageOutputName())
	envs = append(envs, "APP_PLATFORM_COMPONENT_TYPE="+string(b.component.GetType()))
	if b.component.GetSourceDir() != "" {
		envs = append(envs, "SOURCE_DIR="+b.component.GetSourceDir())
	}

	if b.buildCommandOverride != "" {
		template.Render(b.getLogWriter(), heredoc.Doc(`
			{{success checkmark}} overriding default build command with custom command:
			{{highlight .}}
		`,
		), charm.IndentString(4, b.buildCommandOverride))
		envs = append(envs, "BUILD_COMMAND="+b.buildCommandOverride)
	} else if b.component.GetBuildCommand() != "" {
		envs = append(envs, "BUILD_COMMAND="+b.component.GetBuildCommand())
	}

	if len(b.versioning.Buildpacks) > 0 {
		versioningJSON, err := json.Marshal(b.versioning.Buildpacks)
		if err != nil {
			return nil, fmt.Errorf("computing buildpack versioning: %w", err)
		}
		envs = append(envs, "VERSION_PINNING_LIST="+string(versioningJSON))
	}

	if exists, err := b.imageExists(ctx, b.AppImageOutputName()); err != nil {
		return nil, err
	} else if exists {
		envs = append(envs, "PREVIOUS_APP_IMAGE_URL="+b.AppImageOutputName())
	}

	if b.localCacheDir != "" {
		envs = append(envs, "APP_CACHE_DIR="+cnbCacheDir)
	}

	sort.Strings(envs)

	return envs, nil
}
