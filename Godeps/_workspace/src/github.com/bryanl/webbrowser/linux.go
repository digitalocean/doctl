package webbrowser

import "os/exec"

// RequireXError signifies X windows is running.
type RequireXError struct{}

func (e *RequireXError) Error() string {
	return "requires X"
}

// LinuxOpener is an implementation of Opener for Linux.
type LinuxOpener struct {
	Env []string
}

var _ Opener = &LinuxOpener{}

// Open opens a URL using xdg-open. Returns an error if X is not available.
func (o *LinuxOpener) Open(u string, windowType WindowType, autoRaise bool) error {
	if getEnv("DISPLAY", o.Env) == "" {
		return &RequireXError{}
	}
	cmd := exec.Command("/usr/bin/xdg-open", u)
	return cmd.Run()
}
