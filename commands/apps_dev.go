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
	"github.com/digitalocean/doctl/commands/charm/selection"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/commands/charm/text"
	"github.com/digitalocean/doctl/commands/charm/textbox"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/apps/builder"
	"github.com/digitalocean/doctl/internal/apps/config"
	"github.com/digitalocean/doctl/internal/apps/workspace"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/muesli/termenv"

	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	// AppsDevDefaultEnvFile is the default env file path.
	AppsDevDefaultEnvFile     = ".env"
	appDevConfigFileNamespace = "dev"
)

// AppsDev creates the apps dev command subtree.
func AppsDev() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "dev",
			Aliases: []string{},
			Short:   "[BETA] Display commands for working with App Platform local development.",
			Long: heredoc.Docf(`
				[BETA] Display commands for working with App Platform local development.

				  To get started, run %s.`,
				"`doctl app dev build`",
			),
		},
	}

	cmd.AddCommand(AppsDevConfig())

	build := CmdBuilder(
		cmd,
		RunAppsDevBuild,
		"build [component name]",
		"Build an app component",
		heredoc.Docf(`
			[BETA] Build an app component locally.

			  The component name is optional unless running non-interactively.

			  All command line flags as optional. You may specify flags to be applied to the current build
			  or use the command %s to permanently configure default values.`,
			"`doctl app dev config`",
		),
		Writer,
		aliasOpt("b"),
	)
	build.DisableFlagsInUseLine = true

	AddStringFlag(
		build, doctl.ArgAppSpec,
		"", "",
		`An optional path to an app spec in JSON or YAML format. Default: .do/app.yaml.`,
	)

	AddStringFlag(
		build, doctl.ArgApp,
		"", "",
		"An optional existing app ID. If specified, the app spec will be fetched from the given app.",
	)

	AddStringFlag(
		build, doctl.ArgEnvFile,
		"", "",
		"An optional path to a .env file with overrides for values of app spec environment variables.",
	)

	AddBoolFlag(
		build, doctl.ArgNoCache,
		"", false,
		"Set to disable build caching.",
	)

	AddStringFlag(
		build, doctl.ArgBuildCommand,
		"", "",
		"An optional build command override for local development.",
	)

	AddDurationFlag(
		build, doctl.ArgTimeout,
		"", 0,
		`An optional timeout duration for the build. Valid time units are "s", "m", "h". Example: 15m30s`,
	)

	AddStringFlag(
		build, doctl.ArgRegistry,
		"", os.Getenv("APP_DEV_REGISTRY"),
		"An optional registry name to tag built container images with.",
	)

	return cmd
}

// RunAppsDevBuild builds an app component locally.
func RunAppsDevBuild(c *CmdConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ws, err := appDevWorkspace(c)
	if err != nil {
		if errors.Is(err, workspace.ErrNoGitRepo) {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			template.Print(heredoc.Doc(`
				{{error (print crossmark " could not find git worktree.")}}

				local builds must be run within git repositories. doctl was run in {{muted .}}
				however this directory is not inside a git worktree.

				make sure you run doctl in the correct app directory where your source code is located.
			`), cwd)
			return ErrExitSilently
		}
		return fmt.Errorf("preparing workspace: %w", err)
	}

	template.Print("{{muted pointerRight}} current app dev workspace: {{muted .}}{{nl}}", ws.Context())

	if ws.Config.AppSpec == nil {
		err := appsDevBuildSpecRequired(ws, c.Apps())
		if err != nil {
			return err
		}
		if err := ws.Config.Load(); err != nil {
			return fmt.Errorf("reloading config: %w", err)
		}
	}

	var hasBuildableComponents bool
	_ = godo.ForEachAppSpecComponent(ws.Config.AppSpec, func(c godo.AppBuildableComponentSpec) error {
		hasBuildableComponents = true
		// Returning an error short-circuits the component iteration.
		// We just want to assert that atleast one buildable component spec exists.
		return fmt.Errorf("short-circuit")
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

	if componentSpec.GetType() == godo.AppComponentTypeFunctions {
		template.Print(heredoc.Doc(`

			{{warning (print crossmark " functions builds are coming soon!")}}
			  please use {{highlight "doctl serverless deploy"}} to build functions in the meantime.

		`), nil)
		return fmt.Errorf("not supported")
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
		template.Render(text.Warning, `{{pointerRight}} build caching disabled{{nl}}`, nil)
		err = ws.ClearCacheDir(ctx, componentName)
		if err != nil {
			return err
		}
	}
	err = ws.EnsureCacheDir(ctx, componentName)
	if err != nil {
		return err
	}

	if builder.IsCNBBuild(componentSpec) && ws.Config.CNBBuilderImage != "" {
		template.Render(text.Warning, `{{checkmark}} using custom builder image {{highlight .}}{{nl}}`, ws.Config.CNBBuilderImage)
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
	if ws.Config.Timeout > 0 {
		template.Render(text.Warning, `{{checkmark}} restricting maximum build duration to {{highlight (duration .)}}{{nl}}`, ws.Config.Timeout)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ws.Config.Timeout)
		defer cancel()
	}
	buildingComponentLine := template.String(
		`building {{lower (snakeToTitle .componentSpec.GetType)}} {{highlight .componentSpec.GetName}} {{muted (print "(" .appName ")")}}`,
		map[string]any{
			"componentSpec": componentSpec,
			"appName":       ws.Config.AppSpec.GetName(),
		},
	)
	template.Print(`{{success checkmark}} {{.}}{{nl 2}}`, buildingComponentLine)

	var (
		wg        sync.WaitGroup
		logWriter io.Writer

		// userCanceled indicates whether the context was canceled by user request
		userCanceled bool
	)
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
		var portEnv string
		var portArg string
		if componentSpec.GetType() == godo.AppComponentTypeService {
			svc := componentSpec.(*godo.AppServiceSpec)
			port := 8080
			if svc.HTTPPort != 0 {
				port = int(svc.HTTPPort)
			}
			portEnv = fmt.Sprintf("-e PORT=%d ", port)
			portArg = fmt.Sprintf("-p 8080:%d ", port)
		} else if componentSpec.GetType() == godo.AppComponentTypeStaticSite {
			// static site config is hard-coded in nginx to 8080 currently
			portArg = "-p 8080:8080 "
		}

		tmpl := `
				{{success checkmark}} successfully built {{success .component}} in {{highlight (duration .dur)}}
				{{success checkmark}} created container image {{success .img}}

				{{pointerRight}} push your image to a container registry using {{highlight "docker push"}}
				{{pointerRight}} or run it locally using {{highlight "docker run"}}; for example:

				   {{muted promptPrefix}} {{highlight (printf "docker run %s--rm %s%s" .port_env .port_arg .img)}}`

		if _, ok := componentSpec.(godo.AppRoutableComponentSpec); ok {
			tmpl += `

				then access your component at {{underline "http://localhost:8080"}}`
		}

		template.Buffered(
			textbox.New().Success(),
			heredoc.Doc(tmpl),
			map[string]any{
				"component": componentSpec.GetName(),
				"img":       res.Image,
				"dur":       res.BuildDuration,
				"port_arg":  portArg,
				"port_env":  portEnv,
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
	// The setting is nested under the "dev" namespace, i.e. dev.config.set.dev-config
	// This is needed to prevent a conflict with the base config setting.
	ns := fmt.Sprintf("%s.%s", appDevConfigFileNamespace, cmdConfig.NS)
	devConfigFilePath, err := cmdConfig.Doit.GetString(ns, doctl.ArgAppDevConfig)
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
	template.Print("{{success checkmark}} preparing app dev environment{{nl}}", nil)
	var images []string

	_, isCNB := componentSpec.(godo.AppCNBBuildableComponentSpec)
	dockerComponentSpec, isDocker := componentSpec.(godo.AppDockerBuildableComponentSpec)

	if isCNB && (!isDocker || dockerComponentSpec.GetDockerfilePath() == "") {
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

	// TODO: ImageExists can be slow. Look into batch fetching all images at once.
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
				// clear the current line
				termenv.ClearLine()
				fmt.Printf("%s%s",
					// move the cursor back to the beginning of the line
					"\r",
					// print the current bar
					charm.IndentString(2, text.Muted.S(jm.Progress.String())),
				)
			}
		}
		// clear the current line
		termenv.ClearLine()
		fmt.Printf("%s%s",
			// move the cursor back to the beginning of the line
			"\r",
			// overwrite the latest progress bar with a success message
			template.String(`{{success checkmark}} done{{nl}}`, nil),
		)
	}
	return nil
}

func appsDevBuildSpecRequired(ws *workspace.AppDev, appsService do.AppsService) error {
	template.Print(heredoc.Doc(`
		{{error (print crossmark " no app spec found.")}}
		  an app spec is required to start a local build. make sure doctl is run in the correct directory where your app code is.

		`,
	), nil)

	options := struct {
		BringAppSpec string
		LinkApp      string
		Chdir        string
	}{
		BringAppSpec: template.String(`i will place an app spec at {{highlight ".do/app.yaml"}}`, nil),
		LinkApp:      "i would like to link an app from my DigitalOcean cloud account and use its app spec",
		Chdir:        "i ran doctl in the wrong directory",
		// TODO: add support for app detection
		// DetectApp: "i'm in my app project directory, auto-detect an app spec for me",
	}
	sel := selection.New(
		[]string{options.BringAppSpec, options.LinkApp, options.Chdir},
		selection.WithFiltering(false),
	)
	opt, err := sel.Select()
	if err != nil {
		return err
	}
	fmt.Print("\n")

	switch opt {
	case options.BringAppSpec:
		template.Print(`place an app spec at {{highlight ".do/app.yaml"}} and re-run doctl.{{nl}}`, nil)
		return ErrExitSilently
	case options.Chdir:
		template.Print(`cd to the correct directory and re-run doctl.{{nl}}`, nil)
		return ErrExitSilently
	case options.LinkApp:
		app, err := appsDevSelectApp(appsService)
		if err != nil {
			return err
		}

		choice, err := confirm.New(
			template.String(
				`link app {{highlight .app.GetSpec.GetName}} to app dev workspace at {{highlight .context}}?`,
				map[string]any{
					"context": ws.Context(),
					"app":     app,
				},
			),
			confirm.WithDefaultChoice(confirm.Yes),
		).Prompt()
		if err != nil {
			return err
		}
		if choice != confirm.Yes {
			return fmt.Errorf("canceled")
		}

		template.Print("{{success checkmark}} linking app {{highlight .}} to dev workspace{{nl}}", app.GetSpec().GetName())

		if err := ws.Config.SetLinkedApp(app); err != nil {
			return fmt.Errorf("linking app: %w", err)
		}
		if err := ws.Config.Write(); err != nil {
			return fmt.Errorf("writing app link to config: %w", err)
		}
		return nil
	}

	return errors.New("unrecognized option")
}

func appsDevSelectApp(appsService do.AppsService) (*godo.App, error) {
	// TODO: consider updating the list component to accept an itemsFunc and displays its own loading screen
	template.Print(`listing apps on your account...`, nil)
	apps, err := appsService.List(false)
	if err != nil {
		return nil, fmt.Errorf("listing apps: %w", err)
	}
	// clear and reset the "listing apps..." line
	termenv.ClearLine()
	fmt.Print("\r")

	listItems := make([]appListItem, len(apps))
	for i, app := range apps {
		listItems[i] = appListItem{app}
	}
	ll := list.New(list.Items(listItems))
	ll.Model().Title = "select an app"
	ll.Model().SetStatusBarItemName("app", "apps")
	selected, err := ll.Select()
	if err != nil {
		return nil, err
	}
	return selected.(appListItem).App, nil
}
