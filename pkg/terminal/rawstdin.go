package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"golang.org/x/term"
)

// ReadRawStdin sets the terminal to raw mode and reads from stdin, sending
// each complete UTF-8 rune to the provided channel. Bytes still arrive one at
// a time, but a multibyte character is buffered across reads until it forms a
// full rune before it is sent: sending it byte-by-byte produces strings that
// are not valid UTF-8 on their own, and json.Marshal silently replaces each
// one with U+FFFD downstream (doctl#2053).
func (t *terminal) ReadRawStdin(ctx context.Context, stdinCh chan<- string) (restoreTerminalFn func(), err error) {
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("error setting terminal to raw mode: %v", err)
	}
	restoreTerminalFn = func() {
		term.Restore(int(os.Stdin.Fd()), oldState) // Restore terminal on exit
	}

	go readRuneBufferedStdin(ctx, os.Stdin, stdinCh)
	return
}

// readRuneBufferedStdin reads from r one byte at a time and sends each
// complete UTF-8 rune to stdinCh as its own string. An invalid leading byte is
// also "full" by FullRune's own definition (it can never become valid by
// appending more bytes), so this never stalls on malformed input; the
// utf8.UTFMax cap is a defensive backstop in case that ever changes, since
// FullRune already resolves by 4 bytes either way.
func readRuneBufferedStdin(ctx context.Context, r io.Reader, stdinCh chan<- string) {
	var buf []byte
	for {
		var b [1]byte
		_, err := r.Read(b[:]) // Read one byte at a time
		if err != nil {
			continue
		}
		buf = append(buf, b[0])

		if !utf8.FullRune(buf) && len(buf) < utf8.UTFMax {
			continue
		}
		send := string(buf)
		buf = buf[:0]

		select {
		case stdinCh <- send:
		case <-ctx.Done():
			return
		}
	}
}
