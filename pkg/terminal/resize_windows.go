package terminal

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

// MonitorResizeEvents monitors the terminal for resize events and sends them to the provided channel.
func MonitorResizeEvents(ctx context.Context, resizeEvents chan<- TerminalSize) error {
	var prevTerminalSize TerminalSize

	ticker := time.NewTicker(250 * time.Millisecond)
	for {
		width, height, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("error getting terminal size: %w", err)
		}
		terminalSize := TerminalSize{Width: width, Height: height}
		if terminalSize != prevTerminalSize {
			prevTerminalSize = terminalSize
			resizeEvents <- terminalSize
		}

		// sleep to avoid hot looping
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return nil
		}
	}
}
