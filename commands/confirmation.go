package commands

import (
	"fmt"

	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/commands/charm/template"
)

// AskForConfirm parses and verifies user input for confirmation.
func AskForConfirm(message string) error {
	if !Interactive {
		warn("Requires confirmation. Use the `--force` flag to continue without confirmation.")
		return ErrExitSilently
	}
	choice, err := confirm.New(
		template.String("Are you sure you want to {{.}}", message),
		confirm.WithDefaultChoice(confirm.No),
	).Prompt()
	if err != nil {
		return err
	}

	if choice != confirm.Yes {
		return fmt.Errorf("Invalid user input")
	}

	return nil
}

// AskForConfirmDelete builds a message to ask the user to confirm deleting
// one or multiple resources and then sends it through to AskForConfirm to
// parses and verifies user input.
func AskForConfirmDelete(resourceType string, count int) error {
	message := fmt.Sprintf("delete this %s?", resourceType)
	if count > 1 {
		resourceType = resourceType + "s"
		message = fmt.Sprintf("delete %d %s?", count, resourceType)
	}

	err := AskForConfirm(message)
	if err != nil {
		return err
	}

	return nil
}
