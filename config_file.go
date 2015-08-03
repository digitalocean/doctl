package doit

import (
	"sort"

	"gopkg.in/yaml.v2"
)

type ConfigArgMap map[string]string

type ConfigFile struct {
	contents []byte
	argMap   ConfigArgMap
}

func NewConfigFile(argMap ConfigArgMap, c []byte) *ConfigFile {
	return &ConfigFile{
		argMap:   argMap,
		contents: c,
	}
}

func (cf *ConfigFile) Args() ([]string, error) {
	a := []string{}
	c := map[string]interface{}{}

	err := yaml.Unmarshal(cf.contents, &c)
	if err != nil {
		return nil, err
	}

	mk := []string{}
	for k := range cf.argMap {
		mk = append(mk, k)
	}

	sort.Strings(mk)
	am := cf.argMap
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

func GlobalArgs(osArgs, newArgs []string) []string {
	return append(osArgs[:1], append(newArgs, osArgs[1:]...)...)
}
