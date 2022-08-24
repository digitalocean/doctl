package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/commands/charm/list"
	"github.com/digitalocean/doctl/commands/charm/pager"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/commands/charm/text"
	"github.com/digitalocean/doctl/commands/charm/textbox"
	"github.com/digitalocean/doctl/internal/apps/builder"
	"github.com/digitalocean/doctl/internal/apps/config"

	"github.com/digitalocean/godo"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

const (
	// AppsDevDefaultSpecPath is the default spec path for an app.
	AppsDevDefaultSpecPath = ".do/app.yaml"
	// AppsDevDefaultEnvFile is the default env file path.
	AppsDevDefaultEnvFile = ".env"
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

	AddBoolFlag(
		build, doctl.ArgNoCache,
		"", false,
		"Whether or not to omit the cache for the build.",
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
		build, doctl.ArgRegistry,
		"", os.Getenv("APP_DEV_REGISTRY"),
		"Registry name to build use for the component build.",
	)

	return cmd
}

// RunAppsDevBuild builds an app component locally.
func RunAppsDevBuild(c *CmdConfig) error {
	ctx := context.Background()

	conf, err := newAppDevConfig(c)
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	var (
		// doctlConfig is the CLI config source.
		doctlConfig = config.DoctlConfigSource(c.Doit, c.NS)
		// appsDevConfig is the dev-config.yaml config source.
		appsDevConfig = appsDevFlagConfigCompat(conf)
		// globalConfig contains global config vars with cli flags as the first priority.
		globalConfig = config.Multi(
			doctlConfig,
			appsDevConfig,
		)
	)

	timeout := globalConfig.GetDuration(doctl.ArgTimeout)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var (
		spec          *godo.AppSpec
		cnbVersioning *godo.AppBuildConfigCNBVersioning
	)
	appID := globalConfig.GetString(doctl.ArgApp)

	// TODO: if this is the user's first time running dev build, ask them if they'd like to
	// link an existing app.
	if appID != "" {
		template.Print(`{{success checkmark}} fetching app details{{nl}}`, AppsDevDefaultSpecPath)
		app, err := c.Apps().Get(appID)
		if err != nil {
			return err
		}
		spec = app.Spec
		cnbVersioning = app.GetBuildConfig().GetCNBVersioning()
	}

	appSpecPath := globalConfig.GetString(doctl.ArgAppSpec)
	if spec == nil && appSpecPath == "" && fileExists(AppsDevDefaultSpecPath) {
		appSpecPath = AppsDevDefaultSpecPath
		template.Print(`{{success checkmark}} using app spec at {{highlight .}}{{nl}}`, AppsDevDefaultSpecPath)
	}
	if appSpecPath != "" {
		spec, err = readAppSpec(os.Stdin, appSpecPath)
		if err != nil {
			return err
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
	if component == "" {
		var components []list.Item
		_ = godo.ForEachAppSpecComponent(spec, func(c godo.AppBuildableComponentSpec) error {
			components = append(components, componentListItem{c})
			return nil
		})

		if len(components) == 1 {
			component = components[0].(componentListItem).spec.GetName()
		} else if len(components) > 1 && Interactive {
			list := list.New(components)
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
		stat, err := os.Stat(conf.ContextPath(sd))
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

	var (
		// componentConfig is the per-component config source.
		componentConfig = appsDevFlagConfigCompat(conf.Components(component))
		// componentGlobalConfig is per-component config that can be overridden at the global/app level as well as via cli flags.
		componentGlobalConfig = config.Multi(
			doctlConfig,
			componentConfig,
			appsDevFlagConfigCompat(conf),
		)
		// componentArgConfig is per-component config that can be overridden via cli flags.
		componentArgConfig = config.Multi(
			doctlConfig,
			componentConfig,
		)
	)

	var envs map[string]string
	envFile := componentGlobalConfig.GetString(doctl.ArgEnvFile)
	if envFile == "" && Interactive && fileExists(conf.ContextPath(AppsDevDefaultEnvFile)) {
		choice, err := confirm.New(
			template.String(`{{highlight .}} exists, use it for env var values?`, AppsDevDefaultEnvFile),
			confirm.WithDefaultChoice(confirm.No),
		).Prompt()
		if err != nil {
			return err
		}
		if choice == confirm.Yes {
			envFile = conf.ContextPath(AppsDevDefaultEnvFile)
		}
	} else if envFile != "" {
		envFile = conf.ContextPath(envFile)
	}
	if envFile != "" {
		envs, err = godotenv.Read(envFile)
		if err != nil {
			return fmt.Errorf("reading env file: %w", err)
		}
	}

	registryName := globalConfig.GetString(doctl.ArgRegistry)
	noCache := globalConfig.GetBool(doctl.ArgNoCache)
	if noCache {
		template.Render(text.Warning, `{{crossmark}} build caching disabled{{nl}}`, nil)
		err = conf.ClearCacheDir(ctx, component)
		if err != nil {
			return err
		}
	}
	err = conf.EnsureCacheDir(ctx, component)
	if err != nil {
		return err
	}

	// if Interactive {
	// 	choice, err := confirm.New(
	// 		"start build?",
	// 		confirm.WithDefaultChoice(confirm.Yes),
	// 		confirm.WithPersistPrompt(confirm.PersistPromptIfNo),
	// 	).Prompt()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if choice != confirm.Yes {
	// 		return fmt.Errorf("cancelled")
	// 	}
	// }

	cli, err := c.Doit.GetDockerEngineClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	buildingComponentLine := template.String(
		`building {{lower (snakeToTitle .GetType)}} {{highlight .GetName}}`,
		componentSpec,
	)
	template.Print(`{{success checkmark}} {{.}}{{nl 2}}`, buildingComponentLine)

	var (
		wg        sync.WaitGroup
		logWriter io.Writer
	)
	if Interactive {
		logPager, err := pager.New(
			pager.WithTitle(buildingComponentLine),
		)
		if err != nil {
			return fmt.Errorf("starting log pager: %w", err)
		}
		wg.Add(1)
		go func() {
			defer cancel()
			defer wg.Done()
			err := logPager.Start(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "pager error: %v\n", err)
			}
		}()
		logWriter = logPager
	} else {
		logWriter = os.Stdout
	}

	var res builder.ComponentBuilderResult
	err = func() error {
		defer cancel()

		builder, err := c.componentBuilderFactory.NewComponentBuilder(cli, conf.ContextDir(), spec, builder.NewBuilderOpts{
			Component:            component,
			LocalCacheDir:        conf.CacheDir(component),
			NoCache:              noCache,
			Registry:             registryName,
			EnvOverride:          envs,
			BuildCommandOverride: componentArgConfig.GetString(doctl.ArgAppDevBuildCommand),
			LogWriter:            logWriter,
			Versioning:           builder.Versioning{CNB: cnbVersioning},
		})
		if err != nil {
			return err
		}
		res, err = builder.Build(ctx)
		if err != nil {
			_, isSnap := os.LookupEnv("SNAP")
			if errors.Is(err, fs.ErrPermission) && isSnap {
				template.Buffered(
					textbox.New().Warning().WithOutput(logWriter),
					`Using the doctl Snap? Grant doctl access to Docker by running {{highlight "sudo snap connect doctl:app-dev-build docker:docker-daemon"}}`,
					nil,
				)
				return err
			}
			return err
		}
		return nil
	}()
	// allow the pager to exit cleanly
	wg.Wait()

	// TODO: differentiate between user-initiated cancel and cancel due to build failure
	// if err == nil {
	// 	err = ctx.Err()
	// 	if errors.Is(err, context.Canceled) {
	// 		err = fmt.Errorf("cancelled")
	// 	}
	// }

	if err != nil {
		return err
	} else if res.ExitCode == 0 {
		template.Buffered(
			textbox.New().Success(),
			`{{success checkmark}} successfully built {{success .img}} in {{highlight (duration .dur)}}`,
			map[string]any{
				"img": res.Image,
				"dur": res.BuildDuration,
			},
		)
	} else {
		template.Buffered(
			textbox.New().Error(),
			`{{error crossmark}} build container exited with code {{error .code}} after {{highlight (duration .dur)}}`,
			map[string]any{
				"code": res.ExitCode,
				"dur":  res.BuildDuration,
			},
		)
	}
	fmt.Print("\n")
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dashToUnderscore(key string) string {
	return strings.ReplaceAll(key, "-", "_")
}

// appsDevFlagConfigCompat replaces dashes with underscores in the key to keep compatibility with doctl.Arg* keys
// while keeping the config file keys consistent with the app spec naming convention.
// for example: --no-cache on the CLI will map to no_cache in the config file.
func appsDevFlagConfigCompat(cs config.ConfigSource) config.ConfigSource {
	return config.MutatingConfigSource(cs, dashToUnderscore)
}
