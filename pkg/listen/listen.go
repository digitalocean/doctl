package listen

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

// SchemaFunc takes a slice of bytes and returns an io.Reader (See Listener.SchemaFunc)
type SchemaFunc func([]byte) (io.Reader, error)

// Listener implements a ListenerService
type Listener struct {
	// URL is the url for the websocket. The schema should be "wss" or "ws"
	URL *url.URL
	// Token is used for authenticating with the websocket. It will be passed
	// using the "token" query parameter.
	Token string
	// SchemaFunc is a function allowing you to customize the output. For example,
	// this can be useful for unmarshal a JSON message and formatting the output.
	// It should return an io.Reader. If set to nil, the raw message will be outputted.
	SchemaFunc SchemaFunc
	// Out is an io.Writer to output to.
	// doctl hint: this should usually be commands.CmdConfig.Out
	Out io.Writer
	// In is an io.Reader to read from.
	// doctl hint: this should usually be os.Stdin
	In *os.File

	done chan bool
	stop chan bool
}

// ListenerService listens to a websocket connection and outputs to the provided io.Writer
type ListenerService interface {
	Start() error
	Stop()
}

var _ ListenerService = &Listener{}

// NewListener returns a configured Listener
func NewListener(url *url.URL, token string, schemaFunc SchemaFunc, out io.Writer, in *os.File) ListenerService {
	return &Listener{
		URL:        url,
		Token:      token,
		SchemaFunc: schemaFunc,
		Out:        out,
		In:         in,

		done: make(chan bool),
		stop: make(chan bool),
	}
}

// Start makes the websocket connection and writes messages to the io.Writer
func (l *Listener) Start() error {
	if l.Token != "" {
		params := l.URL.Query()
		params.Set("token", l.Token)
		l.URL.RawQuery = params.Encode()
	}

	c, _, err := websocket.DefaultDialer.Dial(l.URL.String(), nil)
	if err != nil {
		return err
	}
	defer c.Close()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	done := l.done
	go func() error {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				return err
			}

			var r io.Reader
			if l.SchemaFunc != nil {
				r, err = l.SchemaFunc(message)
				if err != nil {
					return err
				}
			} else {
				r = bytes.NewReader(message)
			}

			io.Copy(l.Out, r)
		}
	}()

	prevWidth, prevHeight := 0, 0
	resizeTerminal := func() error {
		width, height, err := term.GetSize(0)
		if err != nil {
			return fmt.Errorf("error getting terminal size: %w", err)
		}
		if width == prevWidth && height == prevHeight {
			return nil
		}
		prevWidth = width
		prevHeight = height

		data := struct {
			Op     string `json:"op"`
			Width  uint16 `json:"width"`
			Height uint16 `json:"height"`
		}{
			Op:     "resize",
			Width:  uint16(width),
			Height: uint16(height),
		}
		if err := c.WriteJSON(data); err != nil {
			return fmt.Errorf("error writing to websocket: %w", err)
		}
		return nil
	}

	var keepaliveTickerC <-chan time.Time
	var resizeCheckerC <-chan time.Time
	var stdinCh chan string

	if l.In != nil {
		keepaliveTicker := time.NewTicker(30 * time.Second)
		resizeChecker := time.NewTicker(250 * time.Millisecond)
		keepaliveTickerC = keepaliveTicker.C
		resizeCheckerC = resizeChecker.C
		stdinCh = make(chan string)
		go func() {
			// Set terminal to raw mode
			oldState, err := term.MakeRaw(int(l.In.Fd()))
			if err != nil {
				fmt.Println("Error setting terminal to raw mode:", err)
				return
			}
			defer term.Restore(int(l.In.Fd()), oldState) // Restore terminal on exit

			for {
				var b [1]byte
				_, err := l.In.Read(b[:]) // Read one byte at a time
				if err != nil {
					fmt.Println("Error reading from stdin:", err)
					break
				}

				stdinCh <- string(b[0])
			}
		}()

		if err := resizeTerminal(); err != nil {
			fmt.Println(err)
		}
	}

	writeStdin := func(v string) error {
		data := struct {
			Op   string `json:"op"`
			Data string `json:"data"`
		}{
			Op:   "stdin",
			Data: v,
		}
		if err := c.WriteJSON(data); err != nil {
			return fmt.Errorf("error writing to websocket: %w", err)
		}
		return nil
	}

	for {
		select {
		case b := <-stdinCh:
			if err := writeStdin(string(b[0])); err != nil {
				fmt.Println("Error writing to websocket:", err)
			}
		case <-resizeCheckerC:
			if err := resizeTerminal(); err != nil {
				fmt.Println(err)
			}
		case <-keepaliveTickerC:
			if err := writeStdin(string("")); err != nil {
				fmt.Println("Error writing to websocket:", err)
			}
			// if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
			// 	return err
			// }
		case <-done:
			return nil
		case <-interrupt:
			return writeCloseMessage(c)
		case <-l.stop:
			return writeCloseMessage(c)
		}
	}
}

// Stop signals the Listener to close the websocket connection
func (l *Listener) Stop() {
	select {
	case <-l.done:
	default:
		l.stop <- true
	}
}

func writeCloseMessage(c *websocket.Conn) error {
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}

	return nil
}
