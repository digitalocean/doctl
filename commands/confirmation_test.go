package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
