package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/commands/charm/list"
	"github.com/digitalocean/doctl/commands/charm/pager"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/commands/charm/text"
	"github.com/digitalocean/doctl/commands/charm/textbox"
	"github.com/digitalocean/doctl/internal/apps/builder"
	"github.com/digitalocean/doctl/internal/apps/config"
	"github.com/digitalocean/doctl/internal/apps/workspace"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	// AppsDevDefaultEnvFile is the default env file path.
	AppsDevDefaultEnvFile = ".env"
)

// AppsDev creates the apps dev command subtree.
func AppsDev() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "dev",
			Aliases: []string{},
			Short:   "[BETA] Display commands for working with app platform local development.",
			Long:    `[BETA] Display commands for working with app platform local development.`,
			Hidden:  true,
		},
	}

	cmd.AddCommand(AppsDevConfig())

	build := CmdBuilder(
		cmd,
		RunAppsDevBuild,
		"build [component name]",
		"Build an app component",
		heredoc.Doc(`
			[BETA] Build an app component locally.
			
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
		build, doctl.ArgBuildCommand,
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

	ws, err := appDevWorkspace(c)
	if err != nil {
		return fmt.Errorf("preparing workspace: %w", err)
	}

	if ws.Config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ws.Config.Timeout)
		defer cancel()
	}

	// TODO: if this is the user's first time running dev build, ask them if they'd like to
	// link an existing app.
	if ws.Config.AppSpec == nil {
		// TODO(ntate); allow app-detect build to remove requirement
		return errors.New("please place an app spec at .do/app.yaml or link an existing app using the --app flag")
	}

	var hasBuildableComponents bool
	_ = godo.ForEachAppSpecComponent(ws.Config.AppSpec, func(c godo.AppBuildableComponentSpec) error {
		hasBuildableComponents = true
		return fmt.Errorf("stop")
	})
	if !hasBuildableComponents {
		return fmt.Errorf("the specified app spec does not contain any buildable components")
	}

	var componentName string
	if len(c.Args) >= 1 {
		componentName = c.Args[0]
	}
	if componentName == "" {
		var components []list.Item
		_ = godo.ForEachAppSpecComponent(ws.Config.AppSpec, func(c godo.AppBuildableComponentSpec) error {
			components = append(components, componentListItem{c})
			return nil
		})

		if len(components) == 1 {
			componentName = components[0].(componentListItem).spec.GetName()
		} else if len(components) > 1 && Interactive {
			list := list.New(components)
			list.Model().Title = "select a component"
			list.Model().SetStatusBarItemName("component", "components")
			selected, err := list.Select()
			if err != nil {
				return err
			} else if selected == nil {
				return fmt.Errorf("canceled")
			}
			selectedComponent, ok := selected.(componentListItem)
			if !ok {
				return fmt.Errorf("unexpected item type %T", selectedComponent)
			}
			componentName = selectedComponent.spec.GetName()
		}
	}

	if componentName == "" {
		if !Interactive {
			return errors.New("component name is required when running non-interactively")
		}
		return errors.New("component is required")
	}

	component := ws.Config.Components[componentName]
	if component == nil {
		// TODO: add support for building without an app spec via app detection
		return fmt.Errorf("component %s does not exist in app spec", componentName)
	}
	componentSpec, ok := component.Spec.(godo.AppBuildableComponentSpec)
	if !ok {
		return fmt.Errorf("cannot build component %s", componentName)
	}
	if componentSpec.GetSourceDir() != "" {
		sd := componentSpec.GetSourceDir()
		stat, err := os.Stat(ws.Context(sd))
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

	cli, err := c.Doit.GetDockerEngineClient()
	if err != nil {
		return err
	}

	err = appDevPrepareEnvironment(ctx, ws, cli, componentSpec)
	if err != nil {
		_, isSnap := os.LookupEnv("SNAP")
		if isSnap && errors.Is(err, fs.ErrPermission) {
			template.Buffered(
				textbox.New().Warning(),
				`Using the doctl Snap? Grant doctl access to Docker by running {{highlight "sudo snap connect doctl:app-dev-build docker:docker-daemon"}}`,
				nil,
			)
		}

		return fmt.Errorf("preparing build environment: %w", err)
	}

	if component.EnvFile != "" {
		template.Print(`{{success checkmark}} using envs from {{highlight .}}{{nl}}`, component.EnvFile)
	} else if Interactive && fileExists(ws.Context(AppsDevDefaultEnvFile)) {
		// TODO: persist env file path to dev config
		choice, err := confirm.New(
			template.String(`{{highlight .}} exists, use it for env var values?`, AppsDevDefaultEnvFile),
			confirm.WithDefaultChoice(confirm.No),
		).Prompt()
		if err != nil {
			return err
		}
		if choice == confirm.Yes {
			err := component.LoadEnvFile(ws.Context(AppsDevDefaultEnvFile))
			if err != nil {
				return fmt.Errorf("reading env file: %w", err)
			}
		}
	}

	if ws.Config.NoCache {
		template.Render(text.Warning, `{{crossmark}} build caching disabled{{nl}}`, nil)
		err = ws.ClearCacheDir(ctx, componentName)
		if err != nil {
			return err
		}
	}
	err = ws.EnsureCacheDir(ctx, componentName)
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
	// 		return fmt.Errorf("canceled")
	// 	}
	// }

	buildingComponentLine := template.String(
		`building {{lower (snakeToTitle .GetType)}} {{highlight .GetName}}`,
		componentSpec,
	)
	template.Print(`{{success checkmark}} {{.}}{{nl 2}}`, buildingComponentLine)

	var (
		wg        sync.WaitGroup
		logWriter io.Writer

		// userCanceled indicates whether the context was canceled by user request
		userCanceled bool
	)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if Interactive {
		logPager, err := pager.New(
			pager.WithTitle(buildingComponentLine),
			pager.WithTitleSpinner(true),
		)
		if err != nil {
			return fmt.Errorf("creating log pager: %w", err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			err := logPager.Start(ctx)
			if err != nil {
				if errors.Is(err, charm.ErrCanceled) {
					userCanceled = true
				} else {
					fmt.Fprintf(os.Stderr, "pager error: %v\n", err)
				}
			}
		}()
		logWriter = logPager
	} else {
		logWriter = os.Stdout
		// In interactive mode, the pager handles ctrl-c. Here, we handle it manually instead.
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				if userCanceled {
					template.Print(
						`{{nl}}{{error (print crossmark " forcing unclean exit")}}{{nl}}`,
						nil,
					)
					os.Exit(1)
				}

				cancel()
				userCanceled = true
				template.Print(
					`{{nl}}{{error (print crossmark " got ctrl-c, cancelling. hit ctrl-c again to force exit.")}}{{nl}}`,
					nil,
				)
			}
		}()
	}

	var res builder.ComponentBuilderResult
	err = func() error {
		defer cancel()

		builder, err := c.componentBuilderFactory.NewComponentBuilder(cli, ws.Context(), ws.Config.AppSpec, builder.NewBuilderOpts{
			Component:               componentName,
			LocalCacheDir:           ws.CacheDir(componentName),
			NoCache:                 ws.Config.NoCache,
			Registry:                ws.Config.Registry,
			EnvOverride:             component.Envs,
			BuildCommandOverride:    component.BuildCommand,
			CNBBuilderImageOverride: ws.Config.CNBBuilderImage,
			LogWriter:               logWriter,
			Versioning:              builder.Versioning{CNB: ws.Config.App.GetBuildConfig().GetCNBVersioning()},
		})
		if err != nil {
			return err
		}
		res, err = builder.Build(ctx)
		if err != nil {
			_, isSnap := os.LookupEnv("SNAP")
			if isSnap && errors.Is(err, fs.ErrPermission) {
				template.Buffered(
					textbox.New().Warning().WithOutput(logWriter),
					`Using the doctl Snap? Grant doctl access to Docker by running {{highlight "sudo snap connect doctl:app-dev-build docker:docker-daemon"}}`,
					nil,
				)
			}

			return err
		}
		return nil
	}()
	// allow the pager to exit cleanly
	wg.Wait()

	if err != nil {
		if errors.Is(err, context.Canceled) && userCanceled {
			return fmt.Errorf("canceled")
		}

		return err
	} else if userCanceled {
		return fmt.Errorf("canceled")
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

func fileExists(path ...string) bool {
	_, err := os.Stat(filepath.Join(path...))
	return err == nil
}

func appDevWorkspace(cmdConfig *CmdConfig) (*workspace.AppDev, error) {
	devConfigFilePath, err := cmdConfig.Doit.GetString(cmdConfig.NS, doctl.ArgAppDevConfig)
	if err != nil {
		return nil, err
	}
	doctlConfig := config.DoctlConfigSource(cmdConfig.Doit, cmdConfig.NS)

	return workspace.NewAppDev(workspace.NewAppDevOpts{
		DevConfigFilePath: devConfigFilePath,
		DoctlConfig:       doctlConfig,
		AppsService:       cmdConfig.Apps(),
	})
}

// PrepareEnvironment pulls required images, validates permissions, etc. in preparation for a component build.
func appDevPrepareEnvironment(ctx context.Context, ws *workspace.AppDev, cli builder.DockerEngineClient, componentSpec godo.AppBuildableComponentSpec) error {
	var images []string
	if componentSpec.GetDockerfilePath() == "" {
		// CNB build
		if ws.Config.CNBBuilderImage != "" {
			images = append(images, ws.Config.CNBBuilderImage)
		} else {
			images = append(images, builder.CNBBuilderImage)
		}

		// TODO: get stack run image from builder image md after we pull it, see below
		images = append(images, "digitaloceanapps/apps-run:7858f2c")
	}

	if componentSpec.GetType() == godo.AppComponentTypeStaticSite {
		images = append(images, builder.StaticSiteNginxImage)
	}

	var toPull []string
	for _, ref := range images {
		exists, err := builder.ImageExists(ctx, cli, ref)
		if err != nil {
			return err
		}
		if !exists {
			toPull = append(toPull, ref)
		}
		// TODO pull if image might be stale
	}

	err := pullDockerImages(ctx, cli, toPull)
	if err != nil {
		return err
	}

	// TODO: get stack run image from builder image md
	// builderImage, err := builder.GetImage(ctx, cli, cnbBuilderImage)
	// if err != nil {
	// 	return err
	// }
	// builderImage.Labels["io.buildpacks.builder.metadata"]

	return nil
}

func pullDockerImages(ctx context.Context, cli builder.DockerEngineClient, images []string) error {
	for _, ref := range images {
		template.Print(`{{success checkmark}} pulling container image {{highlight .}}{{nl}}`, ref)

		r, err := cli.ImagePull(ctx, ref, types.ImagePullOptions{})
		if err != nil {
			return fmt.Errorf("pulling container image %s: %w", ref, err)
		}
		defer r.Close()

		dec := json.NewDecoder(r)
		for {
			var jm jsonmessage.JSONMessage
			err := dec.Decode(&jm)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return err
			}

			if jm.Error != nil {
				return jm.Error
			}

			if jm.Aux != nil {
				continue
			}

			if jm.Progress != nil {
				fmt.Printf("\r%s", charm.IndentString(2, text.Muted.S(jm.Progress.String()))) // go back to the start of the line and print the bar
			}
		}
		fmt.Printf("\r%s\n", charm.IndentString(2, template.String(`{{success checkmark}} done`, nil)))
	}
	return nil
}
