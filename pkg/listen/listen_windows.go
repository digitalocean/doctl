package listen

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/term"
)

// MonitorResizeEvents monitors the terminal for resize events and sends them to the provided channel.
func (l *Listener) MonitorResizeEvents(ctx context.Context, fd int, resizeEvents chan<- TerminalSize) error {
	var prevTerminalSize TerminalSize

	ticker := time.NewTicker(250 * time.Millisecond)
	for {
		width, height, err := term.GetSize(fd)
		if err != nil {
			return fmt.Errorf("error getting terminal size: %w", err)
		}
		terminalSize := TerminalSize{Width: width, Height: height}
		if terminalSize != prevTerminalSize {
			prevTerminalSize = terminalSize
			select {
			case resizeEvents <- terminalSize:
			case <-ctx.Done():
				return nil
			}
		}

		// sleep to avoid hot looping
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return nil
		}
	}
}
