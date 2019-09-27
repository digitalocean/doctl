package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// retriveUserInput is a function that can retrive user input in form of string. By default,
// it will prompt the user. In test, you can replace this with code that returns the appropriate response.
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

// AskForConfirm parses and verifies user input for confirmation.
func AskForConfirm(message string) error {
	answer, err := retrieveUserInput(message)
	if err != nil {
		return fmt.Errorf("Unable to parse users input: %s", err)
	}

	if answer != "y" && answer != "ye" && answer != "yes" {
		return fmt.Errorf("Invalid user input")
	}

	return nil
}
