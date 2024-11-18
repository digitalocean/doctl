package terminal

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/term"
)

// ReadRawStdin reads raw stdin.
func ReadRawStdin(ctx context.Context, stdinCh chan<- string) error {
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("error setting terminal to raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState) // Restore terminal on exit

	for {
		var b [1]byte
		_, err := os.Stdin.Read(b[:]) // Read one byte at a time
		if err != nil {
			return fmt.Errorf("error reading stdin: %v", err)
		}

		select {
		case stdinCh <- string(b[:]):
		case <-ctx.Done():
			return nil
		}
	}
}
