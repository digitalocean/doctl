package commands

import "os"

func homeDir() string {
	return os.Getenv("USERPROFILE")
}
