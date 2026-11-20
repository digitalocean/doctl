package terminal

import (
	"context"
	"io"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

// Simulates a raw terminal delivering a multibyte character's bytes one read
// at a time (doctl#2053: each byte, sent as its own string, is not valid
// UTF-8 on its own, so downstream JSON encoding replaces it with U+FFFD).
func TestReadRuneBufferedStdin_AssemblesMultibyteRunes(t *testing.T) {
	const input = "aě€𐍈z" // ascii, 2-byte, 3-byte, 4-byte, ascii
	wantRunes := utf8.RuneCountInString(input)

	pr, pw := io.Pipe()
	defer pr.Close()
	stdinCh := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go readRuneBufferedStdin(ctx, pr, stdinCh)
	go func() {
		for _, b := range []byte(input) {
			pw.Write([]byte{b}) // one byte per read, like the real raw-mode read loop
		}
	}()

	var got string
	for i := 0; i < wantRunes; i++ {
		select {
		case chunk := <-stdinCh:
			if !assert.Truef(t, utf8.ValidString(chunk), "chunk %q is not valid UTF-8 on its own", chunk) {
				return
			}
			got += chunk
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for rune %d/%d", i+1, wantRunes)
		}
	}
	assert.Equal(t, input, got)
}

// A byte that can never start a valid UTF-8 sequence must still flush
// immediately rather than wait forever for continuation bytes that will
// never arrive.
func TestReadRuneBufferedStdin_FlushesInvalidByteWithoutBlocking(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	stdinCh := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go readRuneBufferedStdin(ctx, pr, stdinCh)
	go pw.Write([]byte{0xFF})

	select {
	case chunk := <-stdinCh:
		assert.Equal(t, []byte{0xFF}, []byte(chunk))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the invalid byte to flush")
	}
}

// A genuine partial sequence (bytes that could still become a valid rune)
// must keep buffering rather than flush early — otherwise a multibyte
// character typed slowly would be split right back into the bug this guards
// against.
func TestReadRuneBufferedStdin_WaitsOutGenuinePartialSequence(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	stdinCh := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go readRuneBufferedStdin(ctx, pr, stdinCh)
	// 0xF0 0x90 0x8D is a valid prefix of a 4-byte sequence; the 4th byte
	// (0x88) completes '𐍈'. Nothing should be sent before it arrives.
	go func() {
		for _, b := range []byte{0xF0, 0x90, 0x8D} {
			pw.Write([]byte{b})
		}
		time.Sleep(20 * time.Millisecond)
		pw.Write([]byte{0x88})
	}()

	select {
	case chunk := <-stdinCh:
		assert.Equal(t, "𐍈", chunk)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the completed rune")
	}
}
