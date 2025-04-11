package terminal

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/term"
)

// ReadRawStdin sets the terminal to raw mode and reads from stdin one byte at a time, sending each byte to the provided channel.
func (t *terminal) ReadRawStdin(ctx context.Context, stdinCh chan<- string) (restoreTerminalFn func(), err error) {
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("error setting terminal to raw mode: %v", err)
	}
	restoreTerminalFn = func() {
		term.Restore(int(os.Stdin.Fd()), oldState) // Restore terminal on exit
	}

	go func() {
		for {
			var b [1]byte
			_, err := os.Stdin.Read(b[:]) // Read one byte at a time
			if err != nil {
				continue
			}

			select {
			case stdinCh <- string(b[:]):
			case <-ctx.Done():
				return
			}
		}
	}()
	return
}
