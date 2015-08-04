package doit

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

// ConfigArgMap is map that maps config values to arguments.
type ConfigArgMap map[string]string

// ConfigArgDir is a map of ConfigArgMaps.
type ConfigArgDir map[string]ConfigArgMap

type yamlMap map[interface{}]interface{}

// ConfigFile represents a configuration file's contents and its ConfigArgMap.
type ConfigFile struct {
	contents []byte
	argDir   ConfigArgDir
}

// NewConfigFile creates a ConfigFile.
func NewConfigFile(argDir ConfigArgDir, c []byte) *ConfigFile {
	return &ConfigFile{
		argDir:   argDir,
		contents: c,
	}
}

// Args generates arguments from a ConfigFile.
func (cf *ConfigFile) Args(entry string) ([]string, error) {
	a := []string{}
	c := yamlMap{}

	err := yaml.Unmarshal(cf.contents, &c)
	if err != nil {
		return nil, err
	}

	p := entry
	if len(p) > 0 {
		p = fmt.Sprintf("commands/%s", p)
	}
	m, err := mapPath(c, p)
	if err != nil {
		return nil, err
	}

	am := cf.argDir[entry]
	mk := []string{}
	for k := range am {
		mk = append(mk, k)
	}

	sort.Strings(mk)
	for _, k := range mk {
		v2 := m.(yamlMap)[k]
		arg := "--" + am[k]

		var val string
		switch v2.(type) {
		default:
			val = v2.(string)
			a = append(a, arg, val)
		case bool:
			if v2.(bool) {
				a = append(a, arg)
			}
		case nil:
			// noop
		}

	}

	return a, nil
}

// GlobalArgs creates a new set of arguments by inserting global arguments.
func GlobalArgs(osArgs, newArgs []string) []string {
	return append(osArgs[:1], append(newArgs, osArgs[1:]...)...)
}

// CommandArgs creates a new set of arguments by appending command arguments.
func CommandArgs(osArgs, newArgs []string) []string {
	return append(osArgs, newArgs...)
}

func mapPath(top yamlMap, path string) (interface{}, error) {
	if len(path) == 0 {
		return top, nil
	}

	keys := strings.Split(path, "/")
	for _, key := range keys {
		switch top[key].(type) {
		case nil:
			return nil, fmt.Errorf("invalid path: %s", path)
		default:
			return top[key], nil
		case yamlMap:
			t, ok := top[key]
			if !ok {
				return nil, fmt.Errorf("invalid path: %s", path)
			}
			top = t.(yamlMap)
		}
	}

	return top, nil
}
