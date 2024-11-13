//go:build !windows
// +build !windows

package listen

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// MonitorResizeEvents monitors the terminal for resize events and sends them to the provided channel.
func (l *Listener) MonitorResizeEvents(ctx context.Context, fd int, resizeEvents chan<- TerminalSize) error {
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, unix.SIGWINCH)
	defer signal.Stop(winch)

	for {
		width, height, err := term.GetSize(fd)
		if err != nil {
			return fmt.Errorf("error getting terminal size: %w", err)
		}
		terminalSize := TerminalSize{Width: width, Height: height}

		resizeEvents <- terminalSize

		select {
		case <-winch:
		case <-ctx.Done():
			return nil
		}
	}
}
