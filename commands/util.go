// +build !windows

package commands

import (
	"fmt"
	"os"
	"strconv"
)

func homeDir() string {
	return os.Getenv("HOME")
}

func extractDropletIDs(s []string) ([]int, error) {
	dropletIDs := []int{}

	for _, e := range s {
		i, err := strconv.Atoi(e)
		if err != nil {
			return nil, fmt.Errorf("Provided value [%v] for droplet id is not of type int", e)
		}
		dropletIDs = append(dropletIDs, i)
	}

	return dropletIDs, nil
}
