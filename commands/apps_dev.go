package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/internal/apps/builder"

	"github.com/charmbracelet/bubbles/list"
	"github.com/digitalocean/godo"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

const (
	// AppsDevDefaultSpecPath is the default spec path for an app.
	AppsDevDefaultSpecPath = ".do/app.yaml"
)

// AppsDev creates the apps dev command subtree.
func AppsDev() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "dev",
			Aliases: []string{},
			Short:   "Display commands for working with app platform local development.",
			Long:    `Display commands for working with app platform local development.`,
		},
	}

	cmd.AddCommand(AppsDevConfig())

	build := CmdBuilder(
		cmd,
		RunAppsDevBuild,
		"build [component name]",
		"Build an app component",
		heredoc.Doc(`
			Build an app component locally.
			
			  The component name must be specified as an argument if running non-interactively.`,
		),
		Writer,
		aliasOpt("b"),
		displayerType(&displayers.Apps{}),
	)
	build.DisableFlagsInUseLine = true

	AddStringFlag(
		build, doctl.ArgAppSpec,
		"", "",
		`Path to an app spec in JSON or YAML format. Set to "-" to read from stdin.`,
	)

	AddStringFlag(
		build, doctl.ArgApp,
		"", "",
		"An optional existing app ID. If specified, the app spec will be fetched from the given app.",
	)

	AddStringFlag(
		build, doctl.ArgEnvFile,
		"", "",
		"Additional environment variables to inject into the build.",
	)

	AddStringFlag(
		build, doctl.ArgAppDevBuildCommand,
		"", "",
		"Optional build command override for local development.",
	)

	AddDurationFlag(
		build, doctl.ArgTimeout,
		"", 0,
		"An optional timeout duration for the build",
	)

	AddStringFlag(
		build, doctl.ArgRegistryName,
		"", os.Getenv("APP_DEV_REGISTRY"),
		"Registry name to build use for the component build.",
	)

	return cmd
}

// RunAppsDevBuild builds an app component locally.
func RunAppsDevBuild(c *CmdConfig) error {
	ctx := context.Background()
	if timeout, _ := c.Doit.GetDuration(c.NS, doctl.ArgTimeout); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	conf, err := newAppDevConfig(c)
	if err != nil {
		return err
	}

	var spec *godo.AppSpec
	appID, err := conf.GetString(doctl.ArgApp)
	if err != nil {
		return err
	}

	// TODO: if this is the user's first time running dev build, ask them if they'd like to
	// link an existing app.
	if appID != "" {
		app, err := c.Apps().Get(appID)
		if err != nil {
			return err
		}
		spec = app.Spec
	}

	appSpecPath, err := conf.GetString(doctl.ArgAppSpec)
	if err != nil {
		return err
	}

	if spec == nil {
		if appSpecPath == "" {
			if _, err := os.Stat(AppsDevDefaultSpecPath); err == nil {
				appSpecPath = AppsDevDefaultSpecPath
				charm.TemplatePrint(heredoc.Doc(`
					{{success checkmark}} using app spec at {{highlight .}}{{nl}}`,
				), AppsDevDefaultSpecPath)
			}
		}
		if appSpecPath != "" {
			spec, err = readAppSpec(os.Stdin, appSpecPath)
			if err != nil {
				return err
			}
		}
	}

	if spec == nil {
		// TODO(ntate); allow app-detect build to remove requirement
		return errors.New("app spec is required for component build")
	}

	var hasBuildableComponents bool
	_ = godo.ForEachAppSpecComponent(spec, func(c godo.AppBuildableComponentSpec) error {
		hasBuildableComponents = true
		return fmt.Errorf("stop")
	})
	if !hasBuildableComponents {
		return fmt.Errorf("the specified app spec does not contain any buildable components")
	}

	var component string
	if len(c.Args) >= 1 {
		component = c.Args[0]
	}
	if Interactive && component == "" {
		var components []list.Item
		_ = godo.ForEachAppSpecComponent(spec, func(c godo.AppBuildableComponentSpec) error {
			components = append(components, componentListItem{c})
			return nil
		})
		list := charm.NewList(components)
		list.Fullscreen = true
		list.Model().Title = "select a component"
		list.Model().SetStatusBarItemName("component", "components")
		selected, err := list.Select()
		if err != nil {
			return err
		} else if selected == nil {
			return fmt.Errorf("cancelled")
		}
		selectedComponent, ok := selected.(componentListItem)
		if !ok {
			return fmt.Errorf("unexpected item type %T", selectedComponent)
		}
		component = selectedComponent.spec.GetName()
	}

	if component == "" {
		if !Interactive {
			return errors.New("component name is required when running non-interactively")
		}
		return errors.New("component name is required")
	}

	componentSpec, err := godo.GetAppSpecComponent[godo.AppBuildableComponentSpec](spec, component)
	if err != nil {
		return err
	}
	if componentSpec.GetSourceDir() != "" {
		sd := componentSpec.GetSourceDir()
		stat, err := os.Stat(sd)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("source dir %s does not exist. please make sure you are running doctl in your app directory", sd)
			}
			return fmt.Errorf("checking source dir %s: %w", sd, err)
		}
		if !stat.IsDir() {
			return fmt.Errorf("invalid source dir %s: not a directory", sd)
		}
	}

	var envs map[string]string
	envFile, err := conf.GetString(doctl.ArgEnvFile)
	if err != nil {
		return err
	}
	if envFile != "" {
		envs, err = godotenv.Read(envFile)
		if err != nil {
			return err
		}
	}

	registryName, err := c.Doit.GetString(c.NS, doctl.ArgRegistryName)
	if err != nil {
		return err
	}
	if registryName == "" {
		return errors.New("registry-name is required")
	}

	buildOverrride, err := conf.GetString(doctl.ArgAppDevBuildCommand)
	if err != nil {
		return err
	}

	if Interactive {
		choice, err := confirm.New(
			"start build?",
			confirm.WithDefaultChoice(confirm.Yes),
		).Prompt()
		if err != nil {
			return err
		}
		if choice != confirm.Yes {
			return fmt.Errorf("cancelled")
		}
	}

	cli, err := c.Doit.GetContainerEngineClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg        sync.WaitGroup
		logWriter io.Writer
	)
	if Interactive {
		pager, err := charm.NewPager(
			charm.PagerWithTitle("Building " + component),
		)
		if err != nil {
			return fmt.Errorf("starting log pager: %w", err)
		}
		wg.Add(1)
		go func() {
			defer cancel()
			defer wg.Done()
			err := pager.Start(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "pager error: %v\n", err)
			}
		}()
		logWriter = pager
	} else {
		logWriter = os.Stdout
	}

	charm.TemplatePrint(heredoc.Doc(`
		{{success checkmark}} building {{lower (snakeToTitle .GetType)}} {{highlight .GetName}}{{nl}}{{nl}}`,
	), componentSpec)

	var res builder.ComponentBuilderResult
	err = func() error {
		defer cancel()
		builder, err := c.componentBuilderFactory.NewComponentBuilder(cli, spec, builder.NewBuilderOpts{
			Component:            component,
			Registry:             registryName,
			EnvOverride:          envs,
			BuildCommandOverride: buildOverrride,
			LogWriter:            logWriter,
		})
		if err != nil {
			return err
		}
		res, err = builder.Build(ctx)
		if err != nil {
			return err
		}
		return nil
	}()
	// allow the pager to exit cleanly
	wg.Wait()

	if err != nil {
		return err
	} else if res.ExitCode == 0 {
		charm.TemplateBuffered(
			charm.NewTextBox().Success(),
			`{{success checkmark}} successfully built {{success .img}} in {{warning (duration .dur)}}`,
			map[string]any{
				"img": res.Image,
				"dur": res.BuildDuration,
			},
		)
	} else {
		charm.TemplateBuffered(
			charm.NewTextBox().Error(),
			`{{error crossmark}} build container exited with code {{highlight .code}} after {{warning (duration .dur)}}`,
			map[string]any{
				"code": res.ExitCode,
				"dur":  res.BuildDuration,
			},
		)
	}
	fmt.Print("\n")
	return nil
}
