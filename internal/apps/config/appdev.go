package config

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/spf13/viper"
)

const (
	// DefaultDevConfigFile is the name of the default dev configuration file.
	DefaultDevConfigFile = "dev-config.yaml"

	// nsComponents is the namespace of the component-specific config tree.
	nsComponents = "components"
)

// type appDevUnknownKeyErr struct {
// 	key string
// }

// func (e *appDevUnknownKeyErr) Error() string {
// 	return fmt.Sprintf("unknown key: %s\nvalid keys: %s", e.key, ValidAppDevKeys())
// }

// var validAppDevKeys = map[string]bool{
// 	doctl.ArgApp:                true,
// 	doctl.ArgAppSpec:            true,
// 	doctl.ArgEnvFile:            true,
// 	doctl.ArgRegistry:       true,
// 	doctl.ArgAppDevBuildCommand: true,
// }

// func ValidAppDevKeys() string {
// 	keys := []string{}
// 	for k := range validAppDevKeys {
// 		keys = append(keys, k)
// 	}
// 	sort.Strings(keys)
// 	return strings.Join(keys, ", ")
// }

type AppDev struct {
	contextDir  string
	doctlConfig ConfigSource
	viper       *viper.Viper
}

func (c *AppDev) CacheDir(component string) string {
	return c.ContextPath(".do", "cache", component)
}

func (c *AppDev) EnsureCacheDir(ctx context.Context, component string) error {
	err := os.MkdirAll(c.ContextPath(".do", "cache", component), os.ModePerm)
	if err != nil {
		return err
	}

	return ensureStringInFile(c.ContextPath(".do", ".gitignore"), "/cache")
}

func (c *AppDev) ClearCacheDir(ctx context.Context, component string) error {
	return os.RemoveAll(c.ContextPath(".do", "cache", component))
}

func (c *AppDev) WriteConfig() error {
	return c.viper.WriteConfig()
}

func (c *AppDev) Set(key string, value any) error {
	// if !validAppDevKeys[key] {
	// 	return &appDevUnknownKeyErr{key}
	// }
	c.viper.Set(key, value)
	return nil
}

func (c *AppDev) Components(component string) ConfigSource {
	return NamespacedConfigSource(c, nsKey(nsComponents, component))
}

func (c *AppDev) ContextDir() string {
	return c.contextDir
}

func (c *AppDev) ContextPath(path ...string) string {
	return filepath.Join(append([]string{c.contextDir}, path...)...)
}

func (c *AppDev) IsSet(key string) bool {
	return c.viper.IsSet(key)
}

func (c *AppDev) GetString(key string) string {
	return c.viper.GetString(key)
}

func (c *AppDev) GetBool(key string) bool {
	return c.viper.GetBool(key)
}

func (c *AppDev) GetDuration(key string) time.Duration {
	return c.viper.GetDuration(key)
}

func New(path string) (*AppDev, error) {
	config := &AppDev{
		viper: viper.New(),
	}

	var err error
	config.contextDir, err = os.Getwd()
	if err != nil {
		return nil, err
	}
	gitRoot, err := findTopLevelGitDir(config.contextDir)
	if err != nil && !errors.Is(err, errNoGitRepo) {
		return nil, err
	}
	if gitRoot != "" {
		config.contextDir = gitRoot
	}

	if path == "" {
		configDir := config.ContextPath(".do")
		err = os.MkdirAll(configDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
		path = filepath.Join(configDir, DefaultDevConfigFile)
		if err := ensureStringInFile(path, ""); err != nil {
			return nil, err
		}
		if err := ensureStringInFile(filepath.Join(configDir, ".gitignore"), DefaultDevConfigFile); err != nil {
			return nil, err
		}
	} else if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	config.viper.SetConfigType("yaml")
	config.viper.SetConfigFile(path)

	if err := config.viper.ReadInConfig(); err != nil {
		return nil, err
	}

	return config, nil
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
			return "", errors.New("no git repository found")
		}
		dir = parent
	}
}
