package builder

import (
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestNewBuilderComponent(t *testing.T) {
	builderFactory := DefaultComponentBuilderFactory{}

	t.Run("no component argument provided", func(t *testing.T) {
		_, err := builderFactory.NewComponentBuilder(nil, ".", &godo.AppSpec{
			Services: []*godo.AppServiceSpec{{
				Name: "web",
			}},
		}, NewBuilderOpts{
			Component: "",
		})
		require.ErrorContains(t, err, "component is required")
	})

	t.Run("component does not exist", func(t *testing.T) {
		missingComponent := "missing-component"
		_, err := builderFactory.NewComponentBuilder(nil, ".", &godo.AppSpec{
			Services: []*godo.AppServiceSpec{{
				Name: "web",
			}},
		}, NewBuilderOpts{
			Component: missingComponent,
		})
		require.ErrorContains(t, err, fmt.Sprintf("component %s not found", missingComponent))
	})

	t.Run("dockerfile builder", func(t *testing.T) {
		builder, err := builderFactory.NewComponentBuilder(nil, ".", &godo.AppSpec{
			Services: []*godo.AppServiceSpec{{
				Name:           "web",
				DockerfilePath: ".",
			}},
		}, NewBuilderOpts{
			Component: "web",
		})
		require.NoError(t, err)
		require.IsTypef(t, &DockerComponentBuilder{}, builder, "expected DockerComponentBuilder but was %T", builder)
	})
}

func TestNewBuilderComponent_CNB(t *testing.T) {
	builderFactory := DefaultComponentBuilderFactory{}

	t.Run("happy path", func(t *testing.T) {
		builder, err := builderFactory.NewComponentBuilder(nil, ".", &godo.AppSpec{
			Services: []*godo.AppServiceSpec{{
				Name: "web",
			}},
		}, NewBuilderOpts{
			Component: "web",
		})
		require.NoError(t, err)
		require.IsTypef(t, &CNBComponentBuilder{}, builder, "expected CNBComponentBuilder but was %T", builder)

		cnbBuilder := builder.(*CNBComponentBuilder)
		// no buildpacks in builder opts
		require.Equal(t, CNBVersioning{}, cnbBuilder.versioning)
	})

	t.Run("buildpack versioning", func(t *testing.T) {
		builder, err := builderFactory.NewComponentBuilder(nil, ".", &godo.AppSpec{
			Services: []*godo.AppServiceSpec{{
				Name: "web",
			}},
		}, NewBuilderOpts{
			Component: "web",
			Versioning: Versioning{
				CNB: &godo.AppBuildConfigCNBVersioning{
					Buildpacks: []*godo.Buildpack{
						{
							ID:           "digitalocean/node",
							MajorVersion: 1,
						},
						{
							ID:           "digitalocean/go",
							MajorVersion: 2,
						},
					},
				},
			},
		})
		require.NoError(t, err)
		cnbBuilder, ok := builder.(*CNBComponentBuilder)
		require.True(t, ok, "expected CNBComponentBuilder but was %T", builder)
		require.Equal(t, CNBVersioning{
			Buildpacks: []*Buildpack{
				{
					ID:      "digitalocean/node",
					Version: "1.0.0",
				},
				{
					ID:      "digitalocean/go",
					Version: "2.0.0",
				},
			},
		}, cnbBuilder.versioning)
	})
}
