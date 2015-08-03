package doit

import (
	"sort"

	"gopkg.in/yaml.v2"
)

// ConfigArgMap is map that maps config values to arguments.
type ConfigArgMap map[string]string

// ConfigArgDir is a map of ConfigArgMaps.
type ConfigArgDir map[string]ConfigArgMap

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
	c := map[string]interface{}{}

	err := yaml.Unmarshal(cf.contents, &c)
	if err != nil {
		return nil, err
	}

	argMap := cf.argDir[entry]
	mk := []string{}
	for k := range argMap {
		mk = append(mk, k)
	}

	sort.Strings(mk)
	am := argMap
	for _, k := range mk {
		v2 := c[k]
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
