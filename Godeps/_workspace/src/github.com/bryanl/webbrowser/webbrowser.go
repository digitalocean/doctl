package webbrowser

import (
	"fmt"
	"os"
	"runtime"
)

// UnsupportedOSError signifies an unsupported OS.
type UnsupportedOSError struct {
	OS string
}

func (e *UnsupportedOSError) Error() string {
	return fmt.Sprintf("unsuported OS %s", e.OS)
}

// WindowType is a type of window to open. Currently this is not supported
// but exists to keep the API stable for a future change.
type WindowType string

const (
	// SameWindow is for opening the URL in the same window.
	SameWindow WindowType = "same_window"
	// NewWindow is for opening the URL in a new window.
	NewWindow WindowType = "new_window"
	// NewTab is for opening the URL in a new tab.
	NewTab WindowType = "new_tab"
)

// Opener is the interface that wraps the Open method. It opens a URL
// with specifications for the type of window and if it should be
// autoraised or not.
type Opener interface {
	Open(url string, windowState WindowType, autoRaise bool) error
}

// Open opens a URL in a web browser window.
func Open(u string, windowState WindowType, autoRaise bool) error {
	o := runtime.GOOS

	opener, err := detectBrowsers(o)
	if err != nil {
		return err
	}

	return opener.Open(u, windowState, autoRaise)
}

func detectBrowsers(o string) (Opener, error) {
	switch o {
	case "windows":
		return registerWindows(os.Environ())
	case "linux":
		return registerLinux(os.Environ())
	case "darwin":
		return registerOSX(os.Environ())
	}
	return nil, &UnsupportedOSError{OS: o}
}

func registerOSX(env []string) (Opener, error) {
	return &OSXOpener{Env: env}, nil
}

func registerLinux(env []string) (Opener, error) {
	return &LinuxOpener{Env: env}, nil
}

func registerWindows(env []string) (Opener, error) {
	return &WindowsOpener{Env: env}, nil
}
