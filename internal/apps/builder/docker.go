package builder

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
)

// ErrorLine ...
type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

// ErrorDetail ...
type ErrorDetail struct {
	Message string `json:"message"`
}

// DockerComponentBuilder ...
type DockerComponentBuilder struct {
	baseComponentBuilder
}

// Build ...
func (b *DockerComponentBuilder) Build(ctx context.Context) (ComponentBuilderResult, error) {
	res := ComponentBuilderResult{}
	if b.component == nil {
		return res, errors.New("no component was provided for the build")
	}

	lw := b.getLogWriter()
	if b.buildCommandOverride != "" {
		template.Render(lw,
			`{{warning (print crossmark " build command overrides are ignored for Dockerfile based builds")}}{{nl 2}}`,
			b.buildCommandOverride,
		)
	}

	buildArgs, err := b.getBuildArgs()
	if err != nil {
		return res, fmt.Errorf("configuring environment variables: %w", err)
	}

	buildContext := filepath.Clean(b.component.GetSourceDir())
	buildContext, err = filepath.Rel(".", buildContext)
	if err != nil {
		return res, err
	}
	// TODO Dockerfile must be relative to the source dir.
	// Make it relative and if it's outside the source dir add it to the archive.
	// ref: https://github.com/docker/cli/blob/9400e3dbe8ebd0bede3ab7023f744a8d7f4397d2/cli/command/image/build.go#L280-L286
	start := time.Now()
	tar, err := archive.TarWithOptions(buildContext, &archive.TarOptions{})
	if err != nil {
		return res, err
	}

	opts := types.ImageBuildOptions{
		Dockerfile: b.component.GetDockerfilePath(),
		Tags: []string{
			b.ImageOutputName(),
		},
		BuildArgs: buildArgs,
		NoCache:   b.localCacheDir == "",
	}
	dockerRes, err := b.cli.ImageBuild(ctx, tar, opts)
	if err != nil {
		res.ExitCode = 1
		return res, err
	}
	defer dockerRes.Body.Close()
	print(dockerRes.Body, lw)
	res.Image = b.ImageOutputName()
	res.BuildDuration = time.Since(start)
	res.ExitCode = 0
	return res, nil
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
			/*
				if out.Aux != nil && out.Aux["ID"] != "" {
					fmt.Printf("ID: %s\n", out.Aux["ID"])
				}
			*/
		} else {
			fmt.Fprintln(w, lastLine)
		}
	}

	errLine := &ErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
