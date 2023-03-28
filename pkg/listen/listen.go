package listen

import (
	"bytes"
	"io"
	"net/url"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
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
func NewListener(url *url.URL, token string, schemaFunc SchemaFunc, out io.Writer) ListenerService {
	return &Listener{
		URL:        url,
		Token:      token,
		SchemaFunc: schemaFunc,
		Out:        out,

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

	for {
		select {
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
