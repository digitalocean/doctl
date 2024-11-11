package console

import (
	"time"

	"golang.org/x/term"
)

func MonitorResizeEvents(fd int, resizeEvents chan<- TerminalSize, stop chan struct{}) {
	go func() {
		var prevTerminalSize TerminalSize

		ticker := time.NewTicker(250 * time.Millisecond)
		for {
			width, height, err := term.GetSize(fd)
			if err != nil {
				return
			}
			terminalSize := TerminalSize{Width: width, Height: height}
			if terminalSize != prevTerminalSize {
				prevTerminalSize = terminalSize
				// try to send size
				select {
				case resizeEvents <- terminalSize:
				case <-stop:
					return
				}
			}

			// sleep to avoid hot looping
			select {
			case <-ticker.C:
			case <-stop:
				return
			default:
			}
		}
	}()
}
