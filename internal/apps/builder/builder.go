//go:generate go run github.com/golang/mock/mockgen -source builder.go -package builder -destination builder_mock.go ComponentBuilderFactory ComponentBuilder

package builder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/godo"
)

// ComponentBuilderFactory is the interface for creating a component builder.
type ComponentBuilderFactory interface {
	NewComponentBuilder(ContainerEngineClient, *godo.AppSpec, NewBuilderOpts) (ComponentBuilder, error)
}

// ComponentBuilder is the interface of building one or more components.
type ComponentBuilder interface {
	Build(context.Context) (ComponentBuilderResult, error)
}

// ComponentBuilderResult ...
type ComponentBuilderResult struct {
	Image         string
	BuildDuration time.Duration
	ExitCode      int
}

type baseComponentBuilder struct {
	cli                  ContainerEngineClient
	spec                 *godo.AppSpec
	component            godo.AppBuildableComponentSpec
	registry             string
	envOverrides         map[string]string
	buildCommandOverride string

	logWriter io.Writer
}

func (b baseComponentBuilder) ImageOutputName() string {
	return fmt.Sprintf("%s/%s:dev", b.registry, b.component.GetName())
}

func (b baseComponentBuilder) getLogWriter() io.Writer {
	if b.logWriter == nil {
		return os.Stdout
	}
	return b.logWriter
}

func (b baseComponentBuilder) getEnvMap() map[string]string {
	envs := map[string]string{}
	lw := b.getLogWriter()

	template.Render(lw, heredoc.Doc(`
		{{success checkmark}} configuring build environment variables... {{nl 2}}`,
	), nil)

	if b.spec != nil {
		for _, e := range b.spec.Envs {
			if e.Type == godo.AppVariableType_Secret {
				template.Render(lw, heredoc.Doc(`
					=> Ignoring SECRET variable {{highlight .GetKey}}{{nl}}`,
				), e)
				continue
			}
			if e.Scope != godo.AppVariableScope_RunTime {
				val := e.Value
				envs[e.Key] = val
			}
		}
	}

	for _, e := range b.component.GetEnvs() {
		if e.Type == godo.AppVariableType_Secret {
			template.Render(lw, heredoc.Doc(`
					=> Ignoring SECRET variable {{highlight .GetKey}}{{nl}}`,
			), e)
			continue
		}
		if e.Scope != godo.AppVariableScope_RunTime {
			val := e.Value
			envs[e.Key] = val
		}
	}

	for k, v := range b.envOverrides {
		v := v
		if _, ok := envs[k]; ok {
			template.Render(lw, heredoc.Doc(`
					=> Overwriting {{highlight .}} with provided env value{{nl}}`,
			), k)
		}
		envs[k] = v
	}

	fmt.Fprint(lw, "\n")

	return envs
}

// NewBuilderOpts ...
type NewBuilderOpts struct {
	Component            string
	Registry             string
	EnvOverride          map[string]string
	BuildCommandOverride string
	LogWriter            io.Writer
}

// DefaultComponentBuilderFactory is the standard component builder factory.
type DefaultComponentBuilderFactory struct{}

// NewComponentBuilder returns the correct builder type depending upon the provided
// app and component.
func (f *DefaultComponentBuilderFactory) NewComponentBuilder(cli ContainerEngineClient, spec *godo.AppSpec, opts NewBuilderOpts) (ComponentBuilder, error) {
	// TODO(ntate): handle DetectionBuilder and allow empty component
	if opts.Component == "" {
		return nil, errors.New("component is required")
	}

	component, err := godo.GetAppSpecComponent[godo.AppBuildableComponentSpec](spec, opts.Component)
	if err != nil {
		return nil, err
	}
	if component == nil {
		return nil, fmt.Errorf("component %s does not exist", opts.Component)
	}

	if component.GetDockerfilePath() == "" {
		return &CNBComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli,
				spec,
				component,
				opts.Registry,
				opts.EnvOverride,
				opts.BuildCommandOverride,
				opts.LogWriter,
			},
		}, nil
	}

	return &DockerComponentBuilder{
		baseComponentBuilder: baseComponentBuilder{
			cli,
			spec,
			component,
			opts.Registry,
			opts.EnvOverride,
			opts.BuildCommandOverride,
			opts.LogWriter,
		},
	}, nil
}
