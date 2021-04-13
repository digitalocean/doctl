package displayers

import "fmt"

// BytesToHumanReadibleUnit converts byte input to a human-readable
// form using the largest notation possible in decimal base.
func BytesToHumanReadibleUnit(bytes uint64) string {
	return bytesToHumanReadibleUnit(bytes, 1000, []string{"k", "M", "G", "T", "P", "E"})
}

// BytesToHumanReadibleUnitBinary converts byte input to a human-readable
// form using the largest notation possible in binary base.
func BytesToHumanReadibleUnitBinary(bytes uint64) string {
	return bytesToHumanReadibleUnit(bytes, 1024, []string{"Ki", "Mi", "Gi", "Ti", "Pi", "Ei"})
}

// BytesToHumanReadibleUnit converts byte input to a human-readable
// form using the largest notation possible.
func bytesToHumanReadibleUnit(bytes uint64, baseUnit uint64, units []string) string {
	if bytes < baseUnit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(baseUnit), 0
	for n := bytes / baseUnit; n >= baseUnit; n /= baseUnit {
		div *= baseUnit
		exp++
	}
	return fmt.Sprintf("%.2f %sB", float64(bytes)/float64(div), units[exp])
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}

	return "no"
}
