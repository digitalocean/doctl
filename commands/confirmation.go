package commands

import (
	"fmt"

	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/commands/charm/template"
)

// AskForConfirm parses and verifies user input for confirmation.
func AskForConfirm(message string) error {
	if !Interactive {
		return errConfirmationRequired
	}
	choice, err := confirm.New(
		template.String("Are you sure you want to {{.}}", message),
		confirm.WithDefaultChoice(confirm.No),
	).Prompt()
	if err != nil {
		return err
	}

	if choice != confirm.Yes {
		return errOperationAborted
	}

	return nil
}

// confirmAction resolves whether an action should proceed: force skips the
// prompt entirely, otherwise the user is asked to confirm with message.
// The caller should return the error as-is - whether that's
// errConfirmationRequired (no terminal to ask at, and no --force) or
// errOperationAborted (declined interactively), it is already the right
// thing for checkErr to show, without the call site re-deciding what error
// to report.
func confirmAction(force bool, message string) error {
	if force {
		return nil
	}
	return AskForConfirm(message)
}

// confirmDelete is confirmAction's counterpart for AskForConfirmDelete's
// canned "delete N <resource>?" message.
func confirmDelete(force bool, resourceType string, count int) error {
	if force {
		return nil
	}
	return AskForConfirmDelete(resourceType, count)
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
