package doit

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/codegangsta/cli"
)

var (
	pluginNameRE = regexp.MustCompile("^doit-plugin-[A-Za-z0-9\\-]+$")
)

// PluginList lists all available plugins.
func PluginList(c *cli.Context) {
	for _, p := range pluginPaths() {
		files, _ := ioutil.ReadDir(p)
		for _, f := range files {
			if pluginNameRE.MatchString(f.Name()) {
				fmt.Println(f.Name())
			}
		}
	}
}

func pluginPaths() []string {
	return []string{
		filepath.Join(os.Getenv("GOPATH"), "bin"),
	}
}
