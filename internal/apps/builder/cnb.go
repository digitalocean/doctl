package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/godo"
	dockertypes "github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/archive"
	"github.com/kballard/go-shellquote"
)

const (
	// CNBBuilderImage represents the local cnb builder.
	CNBBuilderImage = "digitaloceanapps/cnb-local-builder:v0.50.4"

	appVarAllowListKey = "APP_VARS"
	appVarPrefix       = "APP_VAR_"
	cnbCacheDir        = "/cnb/cache"
)

var dockerSocketPath = "/var/run/docker.sock"

// CNBComponentBuilder represents a CNB builder.
type CNBComponentBuilder struct {
	baseComponentBuilder
	versioning           CNBVersioning
	localCacheDir        string
	buildContainer       containertypes.ContainerCreateCreatedBody
	builderImageOverride string
	cnbComponent         godo.AppCNBBuildableComponentSpec
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

// Build attempts to build the requested component using the CNB Builder and tags the resulting container images.
func (b *CNBComponentBuilder) Build(ctx context.Context) (res ComponentBuilderResult, err error) {
	if b.component == nil {
		return res, errors.New("no component was provided for the build")
	}

	env, err := b.cnbEnv(ctx)
	if err != nil {
		return res, fmt.Errorf("configuring environment variables: %w", err)
	}

	var sourceDockerSock string
	switch runtime.GOOS {
	case "darwin", "windows":
		// mac/windows docker-for-desktop includes the raw socket in the VM
		sourceDockerSock = "/var/run/docker.sock.raw"
	default:
		sourceDockerSock, err = filepath.EvalSymlinks(dockerSocketPath)
		if err != nil {
			return res, fmt.Errorf("finding docker engine socket: %w", err)
		}
	}

	mounts := []mount.Mount{{
		Type:   mount.TypeBind,
		Source: sourceDockerSock,
		Target: dockerSocketPath,
	}}
	if !b.copyOnWriteSemantics {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: b.contextDir,
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

	b.buildContainer, err = b.cli.ContainerCreate(ctx, &containertypes.Config{
		Image:        b.builderImage(),
		Entrypoint:   []string{"sh", "-c", "sleep infinity"},
		AttachStdout: true,
		AttachStderr: true,
	}, &containertypes.HostConfig{
		Mounts: mounts,
	}, nil, nil, "")
	if err != nil {
		return res, fmt.Errorf("creating build container: %w", err)
	}

	start := time.Now()
	defer func() {
		res.BuildDuration = time.Since(start)
		// we use context.Background() so we can remove the container if the original context is cancelled.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = b.cli.ContainerRemove(ctx, b.buildContainer.ID, dockertypes.ContainerRemoveOptions{
			Force: true,
		})
		b.buildContainer = containertypes.ContainerCreateCreatedBody{}
	}()

	if err := b.cli.ContainerStart(ctx, b.buildContainer.ID, dockertypes.ContainerStartOptions{}); err != nil {
		return res, fmt.Errorf("starting build container: %w", err)
	}

	lw := b.getLogWriter()
	if !b.copyOnWriteSemantics {
		template.Render(lw, "{{success checkmark}} mounting app workspace{{nl}}", nil)
	} else {
		template.Render(lw, heredoc.Doc(`
			{{success checkmark}} copying local app workspace to build container
			  {{muted (print "    local: " .)}}
			  {{muted "container: /workspace"}}
		`), b.contextDir)
		// Prepare source copy info.
		srcInfo, err := archive.CopyInfoSourcePath(b.contextDir, true)
		if err != nil {
			return res, fmt.Errorf("preparing app workspace: %w", err)
		}
		srcArchive, err := archive.TarResource(srcInfo)
		if err != nil {
			return res, fmt.Errorf("preparing app workspace: %w", err)
		}
		defer srcArchive.Close()
		dstInfo := archive.CopyInfo{
			Path:  "/workspace",
			IsDir: true,
		}
		archDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
		if err != nil {
			return res, fmt.Errorf("archiving app workspace: %w", err)
		}
		defer preparedArchive.Close()
		err = b.cli.CopyToContainer(ctx, b.buildContainer.ID, archDir, preparedArchive, dockertypes.CopyToContainerOptions{
			AllowOverwriteDirWithFile: false,
			CopyUIDGID:                false,
		})
		if err != nil {
			return res, fmt.Errorf("copying app workspace to build container: %w", err)
		}
	}

	template.Render(lw, "{{success checkmark}} building{{nl 2}}", nil)
	err = b.runExec(
		ctx,
		b.buildContainer.ID,
		[]string{"sh", "-c", "/.app_platform/build.sh"},
		env,
		b.getLogWriter(),
		nil,
	)
	if err != nil {
		return res, err
	}
	res.Image = b.AppImageOutputName()

	if b.component.GetType() == godo.AppComponentTypeStaticSite {
		err = b.buildStaticSiteImage(ctx)
		if err != nil {
			return res, err
		}
		res.Image = b.StaticSiteImageOutputName()
	}

	return res, nil
}

func (b *CNBComponentBuilder) readFileFromContainer(ctx context.Context, path string) (string, error) {
	var buf bytes.Buffer
	err := b.runExec(
		ctx,
		b.buildContainer.ID,
		[]string{"cat", path},
		nil,
		&buf,
		nil,
	)
	if err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

func (b *CNBComponentBuilder) writeFileToContainer(ctx context.Context, path string, content []byte) error {
	return b.runExec(
		ctx,
		b.buildContainer.ID,
		[]string{"sh", "-c", "cat >" + shellquote.Join(path)},
		nil,
		nil,
		bytes.NewReader(content),
	)
}

func (b *CNBComponentBuilder) buildStaticSiteImage(ctx context.Context) error {
	lw := b.getLogWriter()

	workspacePath, err := b.readFileFromContainer(ctx, "/.app_platform/local/WORKSPACE_PATH")
	if err != nil {
		return err
	}

	assetsPath, err := b.readFileFromContainer(ctx, "/.app_platform/local/ASSETS_PATH")
	if err != nil {
		return err
	}
	assetsPath, err = filepath.Rel(workspacePath, assetsPath)
	if assetsPath == "." {
		assetsPath = "./"
	} else {
		assetsPath = "./" + assetsPath + "/"
	}

	template.Render(lw, `{{success checkmark}} building static site image{{nl 2}}`, nil)

	err = b.writeFileToContainer(ctx, workspacePath+"/nginx.conf", []byte(b.getStaticNginxConfig()))
	if err != nil {
		return fmt.Errorf("writing nginx config: %w", err)
	}

	dockerfile, buildArgs, err := b.staticSiteDockerfile(assetsPath)
	if err != nil {
		return err
	}
	err = b.writeFileToContainer(ctx, workspacePath+"/Dockerfile.static", dockerfile)
	if err != nil {
		return fmt.Errorf("writing static site config: %w", err)
	}

	// build the static site docker image within the build container
	dockerBuildCmd := []string{
		"docker", "build",
		"-t", b.StaticSiteImageOutputName(),
		"-f", workspacePath + "/Dockerfile.static",
	}
	dockerBuildCmd = append(dockerBuildCmd, buildArgsToCmd(buildArgs)...)
	dockerBuildCmd = append(dockerBuildCmd, workspacePath)

	err = b.runExec(
		ctx,
		b.buildContainer.ID,
		[]string{"sh", "-c", shellquote.Join(dockerBuildCmd...)},
		nil,
		lw,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (b *CNBComponentBuilder) staticSiteDockerfile(assetsPath string) (dockerfile []byte, buildArgs map[string]*string, err error) {
	dockerfile = []byte(`
ARG nginx_image
ARG assets_path
FROM ${nginx_image}

COPY ${assets_path} /www
RUN test -f /www/nginx.conf && rm -f /www/nginx.conf

COPY ./nginx.conf /etc/nginx/conf.d/default.conf
`)

	buildArgs = map[string]*string{
		"nginx_image": strPtr(StaticSiteNginxImage),
		"assets_path": &assetsPath,
	}
	return
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
	} else if b.cnbComponent.GetBuildCommand() != "" {
		envs = append(envs, "BUILD_COMMAND="+b.cnbComponent.GetBuildCommand())
	}

	if len(b.versioning.Buildpacks) > 0 {
		versioningJSON, err := json.Marshal(b.versioning.Buildpacks)
		if err != nil {
			return nil, fmt.Errorf("computing buildpack versioning: %w", err)
		}
		envs = append(envs, "VERSION_PINNING_LIST="+string(versioningJSON))
	}

	if exists, err := ImageExists(ctx, b.cli, b.AppImageOutputName()); err != nil {
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

func buildArgsToCmd(buildArgs map[string]*string) []string {
	var (
		cmd  []string
		keys = make([]string, 0, len(buildArgs))
	)
	for k, v := range buildArgs {
		if v != nil {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		cmd = append(cmd, "--build-arg", k+"="+(*buildArgs[k]))
	}
	return cmd
}

func (b *CNBComponentBuilder) builderImage() string {
	if b.builderImageOverride != "" {
		return b.builderImageOverride
	}

	return CNBBuilderImage
}
