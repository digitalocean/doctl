package builder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	// CNBBuilderImage represents the local cnb builder.
	CNBBuilderImage = "digitaloceanapps/cnb-local-builder:v0.46.0"

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

	mounts := []mount.Mount{{
		Type:   mount.TypeBind,
		Source: "/var/run/docker.sock",
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
			return res, err
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

	execRes, err := b.cli.ContainerExecCreate(ctx, buildContainer.ID, types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Env:          env,
		Cmd:          []string{"sh", "-c", "/.app_platform/build.sh"},
	})
	if err != nil {
		return res, err
	}
	defer func() {
		ctx := context.Background()
		execInspectRes, err := b.cli.ContainerExecInspect(ctx, execRes.ID)
		if err == nil {
			res.ExitCode = execInspectRes.ExitCode
		}
	}()

	attachRes, err := b.cli.ContainerExecAttach(ctx, execRes.ID, types.ExecStartCheck{})
	if err != nil {
		return res, err
	}
	defer attachRes.Close()

	// read the output
	outputDone := make(chan error)

	go func() {
		// StdCopy demultiplexes the stream into two buffers
		_, err := stdcopy.StdCopy(b.getLogWriter(), b.getLogWriter(), attachRes.Reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return res, err
		}
		res.Image = b.ImageOutputName()
	case <-ctx.Done():
		return res, ctx.Err()
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
	envs = append(envs, "APP_IMAGE_URL="+b.ImageOutputName())
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

	if exists, err := b.imageExists(ctx, b.ImageOutputName()); err != nil {
		return nil, err
	} else if exists {
		envs = append(envs, "PREVIOUS_APP_IMAGE_URL="+b.ImageOutputName())
	}

	if b.localCacheDir != "" {
		envs = append(envs, "APP_CACHE_DIR="+cnbCacheDir)
	}

	sort.Strings(envs)

	return envs, nil
}
