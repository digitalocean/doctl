package listen

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
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
	// InputCh is a channel to send input to the websocket
	InCh <-chan []byte
}

// ListenerService listens to a websocket connection and outputs to the provided io.Writer
type ListenerService interface {
	Listen(ctx context.Context) error
	ReadRawStdin(ctx context.Context, stdinCh chan<- byte) error
	MonitorResizeEvents(ctx context.Context, fd int, resizeEvents chan<- TerminalSize) error
}

var _ ListenerService = &Listener{}

// NewListener returns a configured Listener
func NewListener(url *url.URL, token string, schemaFunc SchemaFunc, out io.Writer, inCh <-chan []byte) ListenerService {
	return &Listener{
		URL:        url,
		Token:      token,
		SchemaFunc: schemaFunc,
		Out:        out,
		InCh:       inCh,
	}
}

// Listen makes the websocket connection and writes messages to the io.Writer
func (l *Listener) Listen(ctx context.Context) error {
	if l.Token != "" {
		params := l.URL.Query()
		params.Set("token", l.Token)
		l.URL.RawQuery = params.Encode()
	}

	c, _, err := websocket.DefaultDialer.Dial(l.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("error creating websocket connection: %w", err)
	}
	defer c.Close()

	done := make(chan struct{})
	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return nil
				}
				return fmt.Errorf("error reading from websocket: %w", err)
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
	})

	grp.Go(func() error {
		for {
			select {
			case data := <-l.InCh:
				if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
					return fmt.Errorf("error writing to websocket: %w", err)
				}
			case <-ctx.Done():
				if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
					return fmt.Errorf("error writing close message: %w", err)
				}
				return nil
			case <-done:
				return nil
			}
		}
	})
	if err := grp.Wait(); err != nil {
		return err
	}
	return nil
}

// ReadRawStdin reads raw stdin.
func (l *Listener) ReadRawStdin(ctx context.Context, stdinCh chan<- byte) error {
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
		case stdinCh <- b[0]:
		case <-ctx.Done():
			return nil
		default:
		}
	}
}
