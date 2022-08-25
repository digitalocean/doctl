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

	"github.com/digitalocean/doctl/internal/apps/config"
)

const (
	// DefaultDevConfigFile is the name of the default dev configuration file.
	DefaultDevConfigFile = "dev-config.yaml"
)

// NewAppDev creates a new AppDev workspace.
//
// If devConfigFilePath is empty, it defaults to <workspace context>/.do/<DefaultDevConfigFile>.
func NewAppDev(devConfigFilePath string, doctlConfig config.ConfigSource) (*AppDev, error) {
	contextDir, err := findContextDir()
	if err != nil {
		return nil, err
	}

	if devConfigFilePath == "" {
		configDir := filepath.Join(contextDir, ".do")
		err = os.MkdirAll(configDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
		devConfigFilePath = filepath.Join(configDir, DefaultDevConfigFile)
		if err := ensureStringInFile(devConfigFilePath, ""); err != nil {
			return nil, err
		}
		if err := ensureStringInFile(filepath.Join(configDir, ".gitignore"), DefaultDevConfigFile); err != nil {
			return nil, err
		}
	}

	c, err := config.New(devConfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("initializing config: %w", err)
	}

	return &AppDev{
		contextDir: contextDir,
		Config: &AppDevConfig{
			appDevConfig: c,
			doctlConfig:  doctlConfig,
		},
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
func (ws *AppDev) Context(path ...string) string {
	return filepath.Join(append([]string{ws.contextDir}, path...)...)
}

type AppDevConfig struct {
	appDevConfig *config.AppDev
	doctlConfig  config.ConfigSource
}

// Write writes the current dev-config.yaml to disk.
func (c *AppDevConfig) Write() error {
	return c.appDevConfig.WriteConfig()
}

// Doctl returns doctl's CLI config.
func (c *AppDevConfig) Doctl() config.ConfigSource {
	return c.doctlConfig
}

// Global returns the dev-config.yaml config with an optional CLI override.
func (c *AppDevConfig) Global(cliOverride bool) config.ConfigSource {
	var cliConfig config.ConfigSource
	if cliOverride {
		cliConfig = c.Doctl()
	}

	return config.Multi(
		cliConfig,
		appsDevFlagConfigCompat(c.appDevConfig),
	)
}

// Set sets a value in dev-config.yaml.
func (c *AppDevConfig) Set(key string, value any) error {
	return c.appDevConfig.Set(key, value)
}

// Component returns per-component config.
//
// componentOnly: in order of priority:
//		1. CLI config (if requested).
//		2. the component's config.
// componentGlobal: in order of priority:
//		1. CLI config (if requested).
//		2. the component's config.
//		3. global config.
func (c *AppDevConfig) Component(component string, cliOverride bool) (componentOnly, componentGlobal config.ConfigSource) {
	var cliConfig config.ConfigSource
	if cliOverride {
		cliConfig = c.Doctl()
	}

	componentOnly = config.Multi(
		cliConfig,
		c.appDevConfig.Components(component),
	)
	componentGlobal = config.Multi(
		componentOnly,
		c.Global(false), // cliOverride is false because it's already accounted for in componentOnly.
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
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNoGitRepo
		}
		dir = parent
	}
}
