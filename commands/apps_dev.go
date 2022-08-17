package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/internal/apps/builder"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/godo"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	AppsDevDefaultSpecPath = ".do/app.yaml"
)

// AppsDev creates the apps dev command subtree.
func AppsDev() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "dev",
			Aliases: []string{},
			Short:   "Display commands for working with app platform local development.",
			Long:    "...",
		},
	}

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

	link := CmdBuilder(
		cmd,
		RunAppsDevLink,
		"link",
		"Link a repository to an app.",
		`Link a repository to an app.`,
		Writer,
		displayerType(&displayers.Apps{}),
	)

	AddStringFlag(
		link, doctl.ArgAppDevLinkConfig,
		"", "",
		`Path to the app dev link config.`,
	)

	unlink := CmdBuilder(
		cmd,
		RunAppsDevUnLink,
		"unlink",
		"Unlink a repository from an app.",
		`Unlink a repository from an app.`,
		Writer,
		displayerType(&displayers.Apps{}),
	)

	AddStringFlag(
		unlink, doctl.ArgAppDevLinkConfig,
		"", "",
		`Path to the app dev link config.`,
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

	var spec *godo.AppSpec

	// TODO(ntate); cleanup
	appID, err := c.Doit.GetString(c.NS, doctl.ArgApp)
	if err != nil {
		return err
	}
	linkConfigFile, err := c.Doit.GetString(c.NS, doctl.ArgAppDevLinkConfig)
	if err != nil {
		return err
	}
	linkConfig, err := newAppLinkConfig(linkConfigFile, false)
	if err != nil && linkConfigFile != "" {
		return err
	}
	if linkConfig != nil && appID == "" {
		appID = linkConfig.GetString(doctl.ArgApp)
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

	if spec == nil {
		specPath, err := c.Doit.GetString(c.NS, doctl.ArgAppSpec)
		if err != nil {
			return err
		}
		if specPath == "" {
			if _, err := os.Stat(AppsDevDefaultSpecPath); err == nil {
				specPath = AppsDevDefaultSpecPath
				_ = charm.TemplatePrint(heredoc.Doc(`
					{{success checkmark}} using app spec at {{highlight .}}{{nl}}`,
				), AppsDevDefaultSpecPath)
			}
		}
		if specPath != "" {
			spec, err = readAppSpec(os.Stdin, specPath)
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
		if Interactive {
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
	envFile, err := c.Doit.GetString(c.NS, doctl.ArgEnvFile)
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

	cli, err := c.Doit.GetContainerEngineClient()
	if err != nil {
		return err
	}

	pager := newLogPager(context.Background())
	pager.title = "Building " + component
	go func() {
		_ = pager.Start()
		pager.content.WriteTo(os.Stdout)
	}()

	builder, err := c.componentBuilderFactory.NewComponentBuilder(cli, spec, builder.NewBuilderOpts{
		Component: component,
		Registry:  registryName,
		Envs:      envs,
		LogWriter: pager,
	})
	if err != nil {
		return err
	}
	res, err := builder.Build(ctx)
	if err != nil {
		return err
	}

	// fmt.Fprintf(
	// 	charm.NewTextBox().Success(),
	// 	"%s Successfully built %s in %s",
	// 	charm.CheckmarkSuccess,
	// 	charm.TextSuccess.S(res.Image),
	// 	charm.TextSuccess.S(res.BuildDuration.Truncate(time.Second).String()),
	// )

	charm.TemplateBuffered(
		charm.NewTextBox().Success(),
		`{{ success checkmark }} Successfully built {{ success .img }} in {{ warning (duration .dur) }}`,
		map[string]any{
			"img": res.Image,
			"dur": res.BuildDuration,
		},
	)
	return nil
}

func newLogPager(ctx context.Context) *logPager {
	// ctx, cancel := context.WithCancel(ctx)
	return &logPager{
		// ctx:    ctx,
		// cancel: cancel,
		start: time.Now(),
	}
}

type logPager struct {
	title string
	// ctx    context.Context
	// cancel context.CancelFunc
	highPerf bool
	p        *tea.Program
	content  bytes.Buffer
	ready    bool
	viewport viewport.Model
	start    time.Time
}

func (p *logPager) Write(b []byte) (int, error) {
	n, err := p.content.Write(b)
	p.p.Send(msgReload{})
	return n, err
}

func (p *logPager) Close() error {
	if p.p != nil {
		p.p.Send(msgCancel{})
	}
	return nil
}

type msgCancel struct{}
type msgReload struct{}
type msgTick struct{}

func (m *logPager) Init() tea.Cmd {
	return m.timerTick()
}

func (m *logPager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case msgCancel:
		return m, tea.Quit
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = m.highPerf
			m.viewport.SetContent(m.content.String())
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		if m.highPerf {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
	case msgReload:
		m.viewport.SetContent(m.content.String())
		m.viewport.GotoBottom()
		if m.highPerf {
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
	case msgTick:
		return m, m.timerTick()
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *logPager) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m *logPager) headerView() string {
	title := m.titleStyle().Render(m.title)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	line = lipgloss.NewStyle().Foreground(lipgloss.Color("#9B9B9B")).Render(line)
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m *logPager) footerView() string {
	info := m.infoStyle().Render(time.Since(m.start).Truncate(time.Second).String())
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	line = lipgloss.NewStyle().Foreground(lipgloss.Color("#9B9B9B")).Render(line)
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m *logPager) titleStyle() lipgloss.Style {
	b := lipgloss.RoundedBorder()
	b.Right = "├"
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color( /*"#EE6FF8"*/ "#dddddd")).
		BorderStyle(b).
		BorderForeground(lipgloss.Color("#9B9B9B")).
		Padding(0, 1)
}

func (m *logPager) infoStyle() lipgloss.Style {
	b := lipgloss.RoundedBorder()
	b.Left = "┤"
	return m.titleStyle().Copy().BorderStyle(b).Foreground(lipgloss.Color("#dddddd"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (p *logPager) timerTick() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return msgTick{}
	})
}

func (p *logPager) Start() error {
	prog := tea.NewProgram(
		p,
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)
	p.p = prog

	return prog.Start()
}

// RunAppsDevLink links a repo to an app.
func RunAppsDevLink(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return errors.New("app id is required")
	}

	appID := c.Args[0]
	_, err := c.Apps().Get(appID)
	if err != nil {
		return err
	}

	linkConfigFile, err := c.Doit.GetString(c.NS, doctl.ArgAppDevLinkConfig)
	if err != nil {
		return err
	}

	linkConfig, err := newAppLinkConfig(linkConfigFile, true)
	if err != nil {
		return err
	}

	linkConfig.Set(doctl.ArgApp, appID)
	err = linkConfig.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}

// RunAppsDevUnLink unlinks a repo to an app.
func RunAppsDevUnLink(c *CmdConfig) error {
	linkConfigFile, err := c.Doit.GetString(c.NS, doctl.ArgAppDevLinkConfig)
	if err != nil {
		return err
	}

	linkConfig, err := newAppLinkConfig(linkConfigFile, false)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	linkConfig.Set(doctl.ArgApp, "")
	err = linkConfig.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}

type appLinkConfig struct {
	*viper.Viper
}

func newAppLinkConfig(linkFile string, createIfNotExists bool) (*appLinkConfig, error) {
	config := &appLinkConfig{
		viper.New(),
	}

	// attempt to find default link file
	if linkFile == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		gitRoot, err := FindTopLevelGitDir(cwd)
		if err != nil {
			return nil, err
		}
		linkFile = gitRoot + "/.dolocal.yaml"
	}

	if _, err := os.Stat(linkFile); errors.Is(err, os.ErrNotExist) && createIfNotExists {
		if f, err := os.Create(linkFile); err == nil {
			f.Close()
		}
	} else if err != nil {
		return nil, err
	}

	config.SetConfigType("yaml")
	config.SetConfigFile(linkFile)

	if err := config.ReadInConfig(); err != nil {
		return nil, err
	}

	return config, nil
}

// FindTopLevelGitDir ...
func FindTopLevelGitDir(workingDir string) (string, error) {
	dir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no git repository found")
		}
		dir = parent
	}
}
