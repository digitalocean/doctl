// +build !windows

package commands

import (
	"os"
)

func homeDir() string {
	return os.Getenv("HOME")
}
