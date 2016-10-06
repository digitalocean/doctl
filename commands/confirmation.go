package commands

import (
	"bufio"
	"os"
	"strings"
)

func AskForConfirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	warnConfirm("Are you sure you want to " + message + " (y/N) ? ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer = strings.ToLower(strings.Replace(answer, "\n", "", 1))
	return answer == "y" || answer == "ye" || answer == "yes"
}
