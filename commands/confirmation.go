package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var retrieveUserInput = func(message string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	warnConfirm("Are you sure you want to " + message + " (y/N) ? ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.Replace(answer, "\n", "", 1)), nil
}

func AskForConfirm(message string) error {
	answer, err := retrieveUserInput(message)
	if err != nil {
		return fmt.Errorf("unable to parse users input: %s", err)
	}
	if answer == "y" || answer == "ye" || answer == "yes" {
		return nil
	} else {
		return fmt.Errorf("invaild user input")
	}
}
