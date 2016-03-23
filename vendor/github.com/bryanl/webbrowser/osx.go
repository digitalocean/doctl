package webbrowser

import (
	"fmt"
	"os/exec"
	"strings"
)

// OSXOpener is an Opener for Mac OSX.
type OSXOpener struct {
	Env []string
}

var _ Opener = &OSXOpener{}

// Open opens a URL using osascript..
func (o *OSXOpener) Open(u string, windowType WindowType, autoRaise bool) error {
	script := fmt.Sprintf(`open location "%s"`, strings.Replace(u, `"`, "%22", -1))
	cmd := exec.Command("/usr/bin/osascript", "-e", script)
	return cmd.Run()
}
