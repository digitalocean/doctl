package commands

import (
	"fmt"
	"strings"
)

func askForConfirm(message string) bool {
	var answer string
	warn("Are you sure you want to " + message + " (y/N) ? ")
	_, err := fmt.Scanln(&answer)
	if err != nil {
		return false
	}
	return verifyAnswer(answer)
}

func verifyAnswer(answer string) bool {
	if strings.ToLower(answer) == "y" || strings.ToLower(answer) == "yes" {
		return true
	}
	return false
}
