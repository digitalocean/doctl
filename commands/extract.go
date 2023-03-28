package commands

import (
	"fmt"
	"strconv"
)

func extractDropletIDs(s []string) ([]int, error) {
	dropletIDs := make([]int, 0, len(s))

	for _, e := range s {
		i, err := strconv.Atoi(e)
		if err != nil {
			return nil, fmt.Errorf("Provided value [%v] for droplet id is not of type int", e)
		}
		dropletIDs = append(dropletIDs, i)
	}

	return dropletIDs, nil
}
