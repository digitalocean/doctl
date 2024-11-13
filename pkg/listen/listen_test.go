package listen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var (
	upgrader = websocket.Upgrader{}
)

func wsHandler(t *testing.T, recvBuffer *bytes.Buffer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer c.Close()
		i := 0
		finish := 5
		go func() {
			// Read messages from websocket and write to buffer
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					return
				}
				recvBuffer.Write(message)
			}
		}()
		for {
			// Give the Close test a chance to close before any sent
			time.Sleep(time.Millisecond * 10)

			i++
			data := struct {
				Message string `json:"message"`
			}{
				Message: fmt.Sprintf("%d\n", i),
			}
			buf := new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(data)
			require.NoError(t, err)

			err = c.WriteMessage(websocket.TextMessage, buf.Bytes())
			require.NoError(t, err)

			if i == finish {
				break
			}
		}
		err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		require.NoError(t, err)
	}
}

func TestListener(t *testing.T) {
	server := httptest.NewServer(wsHandler(t, nil))
	defer server.Close()

	u := "ws" + strings.TrimPrefix(server.URL, "http")
	url, err := url.Parse(u)
	require.NoError(t, err)

	buffer := &bytes.Buffer{}

	listener := NewListener(url, "", nil, buffer, nil)
	err = listener.Listen(context.Background())
	require.NoError(t, err)

	want := `{"message":"1\n"}
{"message":"2\n"}
{"message":"3\n"}
{"message":"4\n"}
{"message":"5\n"}
`
	require.Equal(t, want, buffer.String())
}

func TestListenerWithSchemaFunc(t *testing.T) {
	server := httptest.NewServer(wsHandler(t, nil))
	defer server.Close()

	u := "ws" + strings.TrimPrefix(server.URL, "http")
	url, err := url.Parse(u)
	require.NoError(t, err)

	buffer := &bytes.Buffer{}

	schemaFunc := func(message []byte) (io.Reader, error) {
		data := struct {
			Message string `json:"message"`
		}{}
		err = json.Unmarshal(message, &data)
		if err != nil {
			return nil, err
		}
		r := strings.NewReader(data.Message)

		return r, nil
	}

	listener := NewListener(url, "", schemaFunc, buffer, nil)
	err = listener.Listen(context.Background())
	require.NoError(t, err)

	want := `1
2
3
4
5
`
	require.Equal(t, want, buffer.String())
}

func TestListenerWithInput(t *testing.T) {
	wsInBuf := &bytes.Buffer{}
	server := httptest.NewServer(wsHandler(t, wsInBuf))
	defer server.Close()

	u := "ws" + strings.TrimPrefix(server.URL, "http")
	url, err := url.Parse(u)
	require.NoError(t, err)

	inputCh := make(chan []byte, 5)
	for i := 0; i < 5; i++ {
		inputCh <- []byte{byte('a' + i)}
	}
	wsOutBuf := &bytes.Buffer{}
	listener := NewListener(url, "", nil, wsOutBuf, inputCh)
	err = listener.Listen(context.Background())
	require.NoError(t, err)

	want := `{"message":"1\n"}
{"message":"2\n"}
{"message":"3\n"}
{"message":"4\n"}
{"message":"5\n"}
`
	require.Equal(t, want, wsOutBuf.String())
	require.Equal(t, "abcde", wsInBuf.String())
}
