//go:build !windows
// +build !windows

package console

import (
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func MonitorResizeEvents(fd int, resizeEvents chan<- TerminalSize, stop chan struct{}) {
	go func() {
		winch := make(chan os.Signal, 1)
		signal.Notify(winch, unix.SIGWINCH)
		defer signal.Stop(winch)

		var prevTerminalSize TerminalSize
		for {
			width, height, err := term.GetSize(fd)
			if err != nil {
				return
			}
			terminalSize := TerminalSize{Width: width, Height: height}
			if terminalSize == prevTerminalSize {
				continue
			}
			prevTerminalSize = terminalSize

			// try to send size
			resizeEvents <- terminalSize

			select {
			case <-winch:
			case <-stop:
				return
			}
		}
	}()
}
