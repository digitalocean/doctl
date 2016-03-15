package doit

import (
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	configFile = ".doctlcfg"
)

// ConfigFile is a doit config file.
type ConfigFile struct {
	location string
}

// NewConfigFile creates an instance of ConfigFile.
func NewConfigFile() (*ConfigFile, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	location := filepath.Join(usr.HomeDir, configFile)

	return &ConfigFile{
		location: location,
	}, nil
}

// Set sets a ConfigFile key to a value. The value should be something
// that serializes to a valid YAML value.
func (cf *ConfigFile) Set(key string, val interface{}) error {
	c, err := cf.Open()
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			err := cf.createConfigFile()
			if err != nil {
				return err
			}

			c, _ = cf.Open()
		default:
			return err
		}

	}

	b, err := ioutil.ReadAll(c)
	if err != nil {
		return err
	}

	var m map[string]interface{}
	err = yaml.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	if m == nil {
		m = map[string]interface{}{}
	}

	m[key] = val

	out, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cf.location, out, 0600)
}

// Open opens a ConfigFile.
func (cf *ConfigFile) Open() (io.Reader, error) {
	_, err := os.Stat(cf.location)
	if err != nil {
		return nil, err
	}

	return os.Open(cf.location)
}

func (cf *ConfigFile) createConfigFile() error {
	f, err := os.Create(cf.location)
	if err != nil {
		return err
	}
	return f.Close()
}
