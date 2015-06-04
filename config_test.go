package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetConfig(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "get-config")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDefaultConfigPath := defaultConfigPath
	defaultConfigPath = filepath.Join(tempDir, "/.docfg")
	defer func() {
		defaultConfigPath = oldDefaultConfigPath
	}()

	var want *Config

	// default config does not exist.
	config, err := getConfig(defaultConfigPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	want = &Config{}
	if !reflect.DeepEqual(config, want) {
		t.Errorf("want %v, got %v", want, config)
	}

	// default config exist with a valid API key.
	configFile, err := os.Create(defaultConfigPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if _, err := configFile.Write([]byte("{\"api_key\": \"apikey\"}")); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	config, err = getConfig(defaultConfigPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	want = &Config{APIKey: "apikey"}

	if !reflect.DeepEqual(config, want) {
		t.Errorf("want %v, got %v", want, config)
	}
}
