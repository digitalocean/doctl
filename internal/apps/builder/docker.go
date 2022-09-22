package builder

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/godo"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
)

// DockerComponentBuilder builds components using a Dockerfile.
type DockerComponentBuilder struct {
	baseComponentBuilder
	dockerComponent godo.AppDockerBuildableComponentSpec
}

// Build executes the component build and tags the resulting container images.
func (b *DockerComponentBuilder) Build(ctx context.Context) (ComponentBuilderResult, error) {
	if b.component == nil {
		return ComponentBuilderResult{}, errors.New("no component was provided for the build")
	}

	if c, ok := b.component.(*godo.AppStaticSiteSpec); ok {
		// NOTE(knasser): we use `path` instead of `filepath` because we want forward slash '/' separated path handling
		// even on windows as the output_dir will be evaluated within the build container which is always linux-based.
		outputDir := path.Clean(c.GetOutputDir())
		if outputDir == "" || outputDir == "." {
			return ComponentBuilderResult{}, errors.New("output_dir is required for dockerfile-based static site builds")
		} else if outputDir == "/" {
			return ComponentBuilderResult{}, errors.New("output_dir may not be /")
		} else if !strings.HasPrefix(outputDir, "/") {
			return ComponentBuilderResult{}, errors.New("output_dir must be an absolute path with dockerfile-based static site builds")
		}
	}

	lw := b.getLogWriter()
	if b.buildCommandOverride != "" {
		template.Render(lw,
			`{{warning (print crossmark " build command overrides are ignored for dockerfile-based builds")}}{{nl}}`,
			b.buildCommandOverride,
		)
	}

	buildArgs, err := b.getBuildArgs()
	if err != nil {
		return ComponentBuilderResult{}, fmt.Errorf("configuring environment variables: %w", err)
	}

	buildContext := b.contextDir
	if sd := filepath.Clean(b.component.GetSourceDir()); sd != "." && sd != "/" {
		buildContext = filepath.Join(buildContext, sd)
	}
	// TODO Dockerfile must be relative to the source dir.
	// Make it relative and if it's outside the source dir add it to the archive.
	// ref: https://github.com/docker/cli/blob/9400e3dbe8ebd0bede3ab7023f744a8d7f4397d2/cli/command/image/build.go#L280-L286
	template.Render(lw, `{{success checkmark}} building image using dockerfile {{highlight .}}{{nl 2}}`, b.dockerComponent.GetDockerfilePath())
	start := time.Now()
	tar, err := archive.TarWithOptions(buildContext, &archive.TarOptions{})
	if err != nil {
		return ComponentBuilderResult{}, fmt.Errorf("preparing build context: %w", err)
	}

	res := ComponentBuilderResult{}
	dockerRes, err := b.cli.ImageBuild(ctx, tar, dockertypes.ImageBuildOptions{
		Dockerfile: b.dockerComponent.GetDockerfilePath(),
		Tags: []string{
			b.AppImageOutputName(),
		},
		BuildArgs: buildArgs,
		NoCache:   b.noCache,
	})
	res.BuildDuration = time.Since(start)
	if err != nil {
		res.ExitCode = 1
		return res, err
	}
	defer dockerRes.Body.Close()
	res.Image = b.AppImageOutputName()
	err = print(dockerRes.Body, lw)
	fmt.Fprint(lw, "\n")
	if err != nil {
		return res, err
	}

	if b.component.GetType() == godo.AppComponentTypeStaticSite {
		err = b.buildStaticSiteImage(ctx)
		res.Image = b.StaticSiteImageOutputName()
		res.BuildDuration = time.Since(start)
		if err != nil {
			return res, err
		}
	}

	return res, nil
}

// buildStaticSiteImage builds a container image that runs a webserver hosting the static site content
func (b *DockerComponentBuilder) buildStaticSiteImage(ctx context.Context) error {
	c, ok := b.component.(*godo.AppStaticSiteSpec)
	if !ok {
		return fmt.Errorf("not a static site component")
	}

	lw := b.getLogWriter()
	template.Render(lw, `{{success checkmark}} building static site image with built assets from {{highlight .}}{{nl 2}}`, c.GetOutputDir())
	tmpDir, err := ioutil.TempDir("", "static-*")
	if err != nil {
		return fmt.Errorf("creating temporary build directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	err = os.WriteFile(filepath.Join(tmpDir, "nginx.conf"), []byte(b.getStaticNginxConfig()), 0664)
	if err != nil {
		return fmt.Errorf("writing nginx config: %w", err)
	}

	dockerfile, buildArgs, err := b.staticSiteDockerfile()
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(tmpDir, "Dockerfile.static"), dockerfile, 0644)
	if err != nil {
		return fmt.Errorf("writing static site config: %w", err)
	}

	tar, err := archive.TarWithOptions(tmpDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("preparing build context: %w", err)
	}
	res, err := b.cli.ImageBuild(ctx, tar, dockertypes.ImageBuildOptions{
		Dockerfile: "./Dockerfile.static",
		Tags:       []string{b.StaticSiteImageOutputName()},
		BuildArgs:  buildArgs,
	})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	err = print(res.Body, lw)
	fmt.Fprint(lw, "\n")
	if err != nil {
		return err
	}
	return nil
}

func (b *DockerComponentBuilder) staticSiteDockerfile() (dockerfile []byte, buildArgs map[string]*string, err error) {
	c, ok := b.component.(*godo.AppStaticSiteSpec)
	if !ok {
		return nil, nil, fmt.Errorf("not a static site component")
	}

	dockerfile = []byte(`
ARG app_image
ARG nginx_image
ARG output_dir
FROM ${app_image} as content
FROM ${nginx_image}
ARG output_dir

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=content ${output_dir}/ /www
`)

	buildArgs = map[string]*string{
		"app_image":   strPtr(b.AppImageOutputName()),
		"nginx_image": strPtr(StaticSiteNginxImage),
		"output_dir":  strPtr(c.GetOutputDir()),
	}
	return
}

func (b *DockerComponentBuilder) getBuildArgs() (map[string]*string, error) {
	envMap, err := b.getEnvMap()
	if err != nil {
		return nil, err
	}
	args := map[string]*string{}

	for k, v := range envMap {
		v := v
		args[k] = &v
	}

	return args, nil
}

type dockerErrorLine struct {
	Error       string                `json:"error"`
	ErrorDetail dockerErrorLineDetail `json:"errorDetail"`
}

type dockerErrorLineDetail struct {
	Message string `json:"message"`
}

type dockerBuildOut struct {
	Stream string
	Aux    map[string]string
}

// TODO(ntate); clean this up and make it nice
func print(rd io.Reader, w io.Writer) error {
	var lastLine string
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()

		out := &dockerBuildOut{}
		if err := json.Unmarshal([]byte(lastLine), out); err == nil {
			fmt.Fprint(w, out.Stream)
		} else {
			fmt.Fprintln(w, lastLine)
		}
	}

	errLine := &dockerErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}
