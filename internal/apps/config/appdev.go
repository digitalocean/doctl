package config

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
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

type AppDev struct {
	viper *viper.Viper
}

func (c *AppDev) WriteConfig() error {
	return c.viper.WriteConfig()
}

func (c *AppDev) Set(key string, value any) error {
	c.viper.Set(key, value)
	return nil
}

func (c *AppDev) Components(component string) ConfigSource {
	return MutatingConfigSource(c, KeyNamespaceMutator(nsKey(nsComponents, component)), nil)
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
	viper := viper.New()
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	viper.SetConfigType("yaml")
	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	return &AppDev{viper}, nil
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
