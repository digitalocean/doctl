//go:generate go run github.com/golang/mock/mockgen -source builder.go -package builder -destination builder_mock.go ComponentBuilderFactory ComponentBuilder

package builder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

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
	cli          ContainerEngineClient
	spec         *godo.AppSpec
	component    godo.AppBuildableComponentSpec
	registry     string
	envOverrides map[string]string

	logWriter io.WriteCloser
}

func (b baseComponentBuilder) ImageOutputName() string {
	return fmt.Sprintf("%s/%s:dev", b.registry, b.component.GetName())
}

// NewBuilderOpts ...
type NewBuilderOpts struct {
	Component string
	Registry  string
	Envs      map[string]string
	LogWriter io.WriteCloser
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
				opts.Envs,
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
			opts.Envs,
			opts.LogWriter,
		},
	}, nil
}
