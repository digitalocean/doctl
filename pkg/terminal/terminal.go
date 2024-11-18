package terminal

import "context"

// Terminal provides an interface for interacting with the terminal
type Terminal interface {
	ReadRawStdin(ctx context.Context, stdinCh chan<- string) (restoreTerminalFn func(), err error)
	MonitorResizeEvents(ctx context.Context, resizeEvents chan<- TerminalSize) error
}

// terminal is an implementation of Terminal
type terminal struct{}

// Ensure terminal implements Terminal
var _ Terminal = &terminal{}

// New returns a new Terminal
func New() Terminal {
	return &terminal{}
}
