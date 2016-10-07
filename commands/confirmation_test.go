package commands

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
