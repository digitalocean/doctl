package webbrowser

import "os/exec"

// WindowsOpener is an Opener for Windows.
type WindowsOpener struct {
	Env []string
}

var _ Opener = &WindowsOpener{}

// Open opens a URL using cmd.exe.
func (o *WindowsOpener) Open(u string, windowType WindowType, autoRaise bool) error {
	cmd := exec.Command("cmd.exe", "/c", "start", u)
	return cmd.Run()
}
