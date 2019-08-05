package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var retrieveUserInput = func(message string) (string, error) {
	return readUserInput(os.Stdin, message)
}

// readUserInput is similar to retrieveUserInput but takes an explicit
// io.Reader to read user input from. It is meant to allow simplified testing
// as to-be-read inputs can be injected conveniently.
func readUserInput(in io.Reader, message string) (string, error) {
	reader := bufio.NewReader(in)
	warnConfirm("Are you sure you want to " + message + " (y/N) ? ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	answer = strings.TrimRight(answer, "\r\n")

	return strings.ToLower(answer), nil
}

func TestAskForConfirmYes(t *testing.T) {
	rui := retrieveUserInput
	defer func() {
		retrieveUserInput = rui
	}()

	retrieveUserInput = func(string) (string, error) {
		return "yes", nil
	}

	err := AskForConfirm("test")
	assert.NoError(t, err)
}

func TestAskForConfirmNo(t *testing.T) {
	rui := retrieveUserInput
	defer func() {
		retrieveUserInput = rui
	}()

	retrieveUserInput = func(string) (string, error) {
		return "no", nil
	}

	err := AskForConfirm("test")
	assert.Error(t, err)
}

func TestAskForConfirmAny(t *testing.T) {
	rui := retrieveUserInput
	defer func() {
		retrieveUserInput = rui
	}()

	retrieveUserInput = func(string) (string, error) {
		return "some-random-message", nil
	}

	err := AskForConfirm("test")
	assert.Error(t, err)
}

func TestAskForConfirmError(t *testing.T) {
	rui := retrieveUserInput
	defer func() {
		retrieveUserInput = rui
	}()

	retrieveUserInput = func(string) (string, error) {
		return "", fmt.Errorf("test-error")
	}

	err := AskForConfirm("test")
	assert.Error(t, err)
}

func TestReadUserInput(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{
			in:   "UNIX\n",
			want: "unix",
		},
		{
			in:   "Windows\r\n",
			want: "windows",
		},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			buf := strings.NewReader(tt.in)
			answer, err := readUserInput(buf, "msg")
			require.NoError(t, err)
			assert.Equal(t, tt.want, answer)
		})
	}
}
