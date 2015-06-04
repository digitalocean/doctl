package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
)

// Config holds the Digital Ocean config.
type Config struct {
	APIKey string `json:"api_key"`
}

func getConfig(configPath string) (*Config, error) {
	c := &Config{}

	data, err := ioutil.ReadFile(configPath)

	// The configuration file is optional so just return an empty config.
	if os.IsNotExist(err) && configPath == defaultConfigPath {
		return c, nil
	}

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("Unable to load configuration file: %s", err)
	}
	return c, nil
}

func getHomeDir() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.HomeDir
}
