package builder

import (
	"context"
	"errors"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	// CNBBuilderImage represents the local cnb builder.
	CNBBuilderImage = "digitaloceanapps/cnb-local-builder:dev"

	appVarAllowListKey = "APP_VARS"
	appVarPrefix       = "APP_VAR_"
)

// CNBComponentBuilder represents a CNB builder
type CNBComponentBuilder struct {
	baseComponentBuilder
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
		Env:          b.cnbEnv(),
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

func (b *CNBComponentBuilder) cnbEnv() []string {
	envMap := b.getEnvMap()
	envs := []string{}

	appVars := []string{}
	for k, v := range envMap {
		envs = append(envs, appVarPrefix+k+"="+v)
		appVars = append(appVars, k)
	}
	if len(appVars) > 0 {
		sort.Strings(appVars)
		envs = append(envs, appVarAllowListKey+"="+strings.Join(appVars, ","))
	}

	envs = append(envs, "APP_IMAGE_URL="+b.ImageOutputName())
	envs = append(envs, "APP_PLATFORM_COMPONENT_TYPE="+string(b.component.GetType()))
	if b.component.GetSourceDir() != "" {
		envs = append(envs, "SOURCE_DIR="+b.component.GetSourceDir())
	}

	if b.buildCommandOverride != "" {
		template.Print(heredoc.Doc(`
				=> Overriding default build command with custom command: {{highlight .}}{{nl}}`,
		), b.buildCommandOverride)
		envs = append(envs, "BUILD_COMMAND="+b.buildCommandOverride)
	} else if b.component.GetBuildCommand() != "" {
		envs = append(envs, "BUILD_COMMAND="+b.component.GetBuildCommand())
	}

	sort.Strings(envs)

	return envs
}
