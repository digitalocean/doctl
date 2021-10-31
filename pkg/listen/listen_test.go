package listen

import (
	"bytes"
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

func wsHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			require.NoError(t, err)
		}
		defer c.Close()
		i := 0
		finish := 5
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
			json.NewEncoder(buf).Encode(data)

			err = c.WriteMessage(websocket.TextMessage, buf.Bytes())
			if err != nil {
				require.NoError(t, err)
			}

			if i == finish {
				break
			}
		}
	}
}

func TestListener(t *testing.T) {
	server := httptest.NewServer(wsHandler(t))
	defer server.Close()

	u := "ws" + strings.TrimPrefix(server.URL, "http")
	url, err := url.Parse(u)
	if err != nil {
		require.NoError(t, err)
	}

	buffer := &bytes.Buffer{}

	listener := NewListener(url, "", nil, buffer)
	err = listener.Start()
	if err != nil {
		require.NoError(t, err)
	}

	want := `{"message":"1\n"}
{"message":"2\n"}
{"message":"3\n"}
{"message":"4\n"}
{"message":"5\n"}
`
	require.Equal(t, want, buffer.String())
}

func TestListenerWithSchemaFunc(t *testing.T) {
	server := httptest.NewServer(wsHandler(t))
	defer server.Close()

	u := "ws" + strings.TrimPrefix(server.URL, "http")
	url, err := url.Parse(u)
	if err != nil {
		require.NoError(t, err)
	}

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

	listener := NewListener(url, "", schemaFunc, buffer)
	err = listener.Start()
	if err != nil {
		t.Fatalf("%v", err)
	}

	want := `1
2
3
4
5
`
	require.Equal(t, want, buffer.String())
}

func TestListenerStop(t *testing.T) {
	server := httptest.NewServer(wsHandler(t))
	defer server.Close()

	u := "ws" + strings.TrimPrefix(server.URL, "http")
	url, err := url.Parse(u)
	if err != nil {
		require.NoError(t, err)
	}

	buffer := &bytes.Buffer{}

	listener := NewListener(url, "", nil, buffer)
	go listener.Start()
	// Stop before any messages have been sent
	listener.Stop()

	require.Equal(t, "", buffer.String())
}
