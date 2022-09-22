package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/apps"
	"github.com/digitalocean/doctl/internal/apps/config"
	"github.com/digitalocean/godo"
	"github.com/joho/godotenv"
)

const (
	// DefaultDevConfigFile is the name of the default dev configuration file.
	DefaultDevConfigFile = "dev-config.yaml"
	// DefaultSpecPath is the default spec path for an app.
	DefaultSpecPath = ".do/app.yaml"

	// SampleDevConfigFile represents a sample dev config file with all options and descriptions.
	SampleDevConfigFile = `
	timeout: duration until command is cancelled
	app: ID of an App Platform App to load the AppSpec from
	spec: Path to the AppSpec to load
	registry: Registry name to use for build image output
	no-cache: Boolean to determine whether or not to use the build cache
	[component-name]:
		build-command: Custom build command override for a given component.
		env-file: Path to an env file to override envs for a given component.
`
)

type NewAppDevOpts struct {
	// DevConfigFilePath is an optional path to the config file. Defaults to <workspace context>/.do/<DefaultDevConfigFile>.
	DevConfigFilePath string
	// DoctlConfig is the doctl CLI config source. Use config.DoctlConfigSource(...) to create it.
	DoctlConfig config.ConfigSource
	// AppsService is the apps API service.
	AppsService do.AppsService
}

// NewAppDev creates a new AppDev workspace.
func NewAppDev(opts NewAppDevOpts) (*AppDev, error) {
	contextDir, err := findContextDir()
	if err != nil {
		return nil, err
	}

	if opts.DevConfigFilePath == "" {
		configDir := filepath.Join(contextDir, ".do")
		err = os.MkdirAll(configDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
		opts.DevConfigFilePath = filepath.Join(configDir, DefaultDevConfigFile)
		if err := ensureStringInFile(opts.DevConfigFilePath, ""); err != nil {
			return nil, err
		}
		if err := ensureStringInFile(filepath.Join(configDir, ".gitignore"), DefaultDevConfigFile); err != nil {
			return nil, err
		}
	}

	appDevConfig, err := config.New(opts.DevConfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("initializing config: %w", err)
	}

	config, err := NewAppDevConfig(appDevConfig, opts.DoctlConfig, opts.AppsService)
	if err != nil {
		return nil, fmt.Errorf("initializing config: %w", err)
	}

	return &AppDev{
		contextDir: contextDir,
		Config:     config,
	}, nil
}

type AppDev struct {
	Config     *AppDevConfig
	contextDir string
}

func (c *AppDev) CacheDir(component string) string {
	return c.Context(".do", "cache", component)
}

func (c *AppDev) EnsureCacheDir(ctx context.Context, component string) error {
	err := os.MkdirAll(c.CacheDir(component), os.ModePerm)
	if err != nil {
		return err
	}

	return ensureStringInFile(c.Context(".do", ".gitignore"), "/cache")
}

func (c *AppDev) ClearCacheDir(ctx context.Context, component string) error {
	return os.RemoveAll(c.CacheDir(component))
}

// Context returns a path relative to the workspace context.
// A call with no arguments returns the workspace context path.
// If an absolute path is given it is returned as-is.
func (ws *AppDev) Context(path ...string) string {
	p := filepath.Join(path...)
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(ws.contextDir, p)
}

type AppDevConfig struct {
	// appSpecPath is the path to the app spec on disk.
	appSpecPath string
	// AppSpec is the app spec for the workspace.
	AppSpec *godo.AppSpec

	// appID is an optional production app id to link the workspace to.
	appID string
	// App is the production app resource if AppID is set.
	App *godo.App
	// AppName is the current spec's or linked app's name.
	// For linked apps: the name is cached and available on subsequent runs before the latest *godo.App is fetched.
	AppName string

	Registry        string
	Timeout         time.Duration
	NoCache         bool
	CNBBuilderImage string

	// Components contains component-specific configuration keyed by component name.
	Components map[string]*AppDevConfigComponent

	appDevConfig *config.AppDev
	doctlConfig  config.ConfigSource
	appsService  do.AppsService
}

type AppDevConfigComponent struct {
	Spec         godo.AppComponentSpec
	EnvFile      string
	Envs         map[string]string
	BuildCommand string
}

// NewAppDevConfig populates an AppDevConfig instance with values sourced from *config.AppDev and doctl.Config.
func NewAppDevConfig(appDevConfig *config.AppDev, doctlConfig config.ConfigSource, appsService do.AppsService) (*AppDevConfig, error) {
	c := &AppDevConfig{
		appDevConfig: appDevConfig,
		doctlConfig:  doctlConfig,
		appsService:  appsService,
	}

	err := c.Load()
	if err != nil {
		return nil, err
	}

	err = c.validate()
	if err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return c, nil
}

func (c *AppDevConfig) SetLinkedApp(app *godo.App) error {
	if err := c.Set("app", app.GetID()); err != nil {
		return err
	}
	if err := c.Set("app_name", app.GetSpec().GetName()); err != nil {
		return err
	}
	return nil
}

// Load loads the config.
//
// Note: the .Components config structure is only loaded for components that are present in the app spec. Configuration
// in dev-config.yaml for components that are not present in the app spec will be ignored.
func (c *AppDevConfig) Load() error {
	// ws - workspace config w/ CLI overrides
	ws := c.workspace(true)

	c.Timeout = ws.GetDuration(doctl.ArgTimeout)
	c.appID = ws.GetString(doctl.ArgApp)
	c.appSpecPath = ws.GetString(doctl.ArgAppSpec)
	c.AppName = ws.GetString("app_name")
	c.Registry = ws.GetString(doctl.ArgRegistry)
	c.NoCache = ws.GetBool(doctl.ArgNoCache)
	c.CNBBuilderImage = ws.GetString("cnb_builder_image")

	err := c.loadAppSpec()
	if err != nil {
		return err
	}

	c.Components = make(map[string]*AppDevConfigComponent)
	_ = godo.ForEachAppSpecComponent(c.AppSpec, func(spec godo.AppBuildableComponentSpec) error {
		name := spec.GetName()
		// component - component config w/ CLI overrides
		// componentWS - component config w/ workspace and CLI overrides
		component, componentWS := c.component(name, true)
		cc := &AppDevConfigComponent{
			Spec:         spec,
			BuildCommand: component.GetString(doctl.ArgBuildCommand),
		}
		cc.LoadEnvFile(componentWS.GetString(doctl.ArgEnvFile))

		c.Components[name] = cc
		return nil
	})

	return nil
}

// LoadEnvFile loads the given file into the component config.
func (c *AppDevConfigComponent) LoadEnvFile(path string) error {
	if path == "" {
		return nil
	}

	envs, err := godotenv.Read(path)
	if err != nil {
		return fmt.Errorf("reading env file: %w", err)
	}

	c.EnvFile = path
	c.Envs = envs
	return nil
}

// loadAppSpec loads the app spec from disk or from godo based on the AppID and AppSpecPath configs.
func (c *AppDevConfig) loadAppSpec() error {
	var err error

	if c.appID != "" {
		template.Print(`{{success checkmark}} fetching app details{{nl}}`, nil)
		c.App, err = c.appsService.Get(c.appID)
		if err != nil {
			return fmt.Errorf("fetching app with id %s: %w", c.appID, err)
		}
		template.Print(`{{success checkmark}} loading config from app {{highlight .}}{{nl}}`, c.App.GetSpec().GetName())
		c.AppSpec = c.App.GetSpec()
	} else if c.appSpecPath == "" && fileExists(DefaultSpecPath) {
		c.appSpecPath = DefaultSpecPath
	}

	if c.appSpecPath != "" {
		tmplPath := c.appSpecPath
		if tmplPath == "-" {
			tmplPath = "stdin"
		}
		template.Print(`{{success checkmark}} using app spec from {{highlight .}}{{nl}}`, tmplPath)
		c.AppSpec, err = apps.ReadAppSpec(os.Stdin, c.appSpecPath)
		if err != nil {
			return err
		}
	}

	if c.AppName != c.AppSpec.GetName() {
		c.AppName = c.AppSpec.GetName()
		c.Set("app_name", c.AppName)
		_ = c.Write() // best effort
	}

	return nil
}

// validate runs validation checks.
func (c *AppDevConfig) validate() error {
	return nil
}

// Write writes the current dev-config.yaml to disk.
// Note that modifying values in the AppDevConfig struct will not affect the contents of the dev-config.yaml file. Instead,
// use the Set(...) method and then call Write().
func (c *AppDevConfig) Write() error {
	return c.appDevConfig.WriteConfig()
}

// doctl returns doctl's CLI config.
func (c *AppDevConfig) doctl() config.ConfigSource {
	return c.doctlConfig
}

// workspace returns the dev-config.yaml config with an optional CLI override.
func (c *AppDevConfig) workspace(cliOverride bool) config.ConfigSource {
	var cliConfig config.ConfigSource
	if cliOverride {
		cliConfig = c.doctl()
	}

	return config.Multi(
		cliConfig,
		appsDevFlagConfigCompat(c.appDevConfig),
	)
}

// Set sets a value in dev-config.yaml.
// Note that the configuration must be reloaded for the new values to be populated in AppDevConfig.
func (c *AppDevConfig) Set(key string, value any) error {
	return c.appDevConfig.Set(key, value)
}

// component returns per-component config.
//
// componentOnly: in order of priority:
//  1. CLI config (if requested).
//  2. the component's config.
//
// componentGlobal: in order of priority:
//  1. CLI config (if requested).
//  2. the component's config.
//  3. global config.
func (c *AppDevConfig) component(component string, cliOverride bool) (componentOnly, componentGlobal config.ConfigSource) {
	var cliConfig config.ConfigSource
	if cliOverride {
		cliConfig = c.doctl()
	}

	componentOnly = config.Multi(
		cliConfig,
		appsDevFlagConfigCompat(c.appDevConfig.Components(component)),
	)
	componentGlobal = config.Multi(
		componentOnly,
		c.workspace(false), // cliOverride is false because it's already accounted for in componentOnly.
	)
	return
}

func ensureStringInFile(file string, val string) error {
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		f, err := os.OpenFile(
			file,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644,
		)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(val)
		return err
	} else if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	if exists, err := regexp.Match(regexp.QuoteMeta(val), b); err != nil {
		return err
	} else if !exists {
		f, err := os.OpenFile(
			file,
			os.O_APPEND|os.O_WRONLY,
			0644,
		)
		if err != nil {
			return err
		}
		defer f.Close()

		if !bytes.HasSuffix(b, []byte("\n")) {
			val = "\n" + val
		}

		_, err = f.WriteString(val)
		return err
	}

	return nil
}

func dashToUnderscore(key string) string {
	return strings.ReplaceAll(key, "-", "_")
}

// appsDevFlagConfigCompat replaces dashes with underscores in the key to keep compatibility with doctl.Arg* keys
// while keeping the config file keys consistent with the app spec naming convention.
// for example: --no-cache on the CLI will map to no_cache in the config file.
func appsDevFlagConfigCompat(cs config.ConfigSource) config.ConfigSource {
	return config.MutatingConfigSource(cs, dashToUnderscore, nil)
}

func findContextDir() (string, error) {
	contextDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	gitRoot, err := findTopLevelGitDir(contextDir)
	if err != nil && !errors.Is(err, errNoGitRepo) {
		return "", err
	}
	if gitRoot != "" {
		contextDir = gitRoot
	}

	return contextDir, nil
}

var errNoGitRepo = errors.New("no git repository found")

// findTopLevelGitDir ...
func findTopLevelGitDir(workingDir string) (string, error) {
	dir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", err
	}

	for {
		if fileExists(dir, ".git"); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNoGitRepo
		}
		dir = parent
	}
}

func fileExists(path ...string) bool {
	_, err := os.Stat(filepath.Join(path...))
	return err == nil
}
