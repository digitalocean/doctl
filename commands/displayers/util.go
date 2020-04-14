package displayers

import "fmt"

const (
	baseUnit = 1000
	units    = "kMGTPE"
)

// BytesToHumanReadibleUnit converts byte input to a human-readable
// form using the largest notation possible.
func BytesToHumanReadibleUnit(bytes uint64) string {
	if bytes < baseUnit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(baseUnit), 0
	for n := bytes / baseUnit; n >= baseUnit; n /= baseUnit {
		div *= baseUnit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), units[exp])
}
