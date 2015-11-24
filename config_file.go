package doit

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	configFile = ".doitcfg"
)

type ConfigFile struct{}

func NewConfigFile() *ConfigFile {
	return &ConfigFile{}
}

func (cf *ConfigFile) Set(key string, val interface{}) error {
	c, err := cf.Open()
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			cf.createConfigFile()
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

	path, err := cf.configFilePath()
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(m)

	return ioutil.WriteFile(path, out, 0600)
}

func (cf *ConfigFile) Open() (io.Reader, error) {
	fp, err := cf.configFilePath()
	if err != nil {
		return nil, fmt.Errorf("can't find home directory: %v", err)
	}
	_, err = os.Stat(fp)
	if err != nil {
		return nil, err
	}

	return os.Open(fp)
}

func (cf *ConfigFile) configFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(usr.HomeDir, configFile)
	return dir, nil
}

func (cf *ConfigFile) createConfigFile() error {
	p, err := cf.configFilePath()
	if err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	return f.Close()
}
