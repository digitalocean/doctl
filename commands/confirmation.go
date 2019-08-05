package commands

import (
	"fmt"
)

// AskForConfirm parses and verifies user input for confirmation.
func AskForConfirm(message string) error {
	answer, err := retrieveUserInput(message)
	if err != nil {
		return fmt.Errorf("unable to parse users input: %s", err)
	}

	if answer != "y" && answer != "ye" && answer != "yes" {
		return fmt.Errorf("invalid user input")
	}

	return nil
}
