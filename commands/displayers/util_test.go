package displayers

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BytesToHumanReadableUnit(t *testing.T) {
	tests := map[string]struct {
		bytes    uint64
		expected string
	}{
		"0 bytes": {
			bytes:    0,
			expected: "0 B",
		},
		"less than one kilobyte": {
			bytes:    uint64(math.Pow10(3)) - 1,
			expected: "999 B",
		},
		"one kilobyte less than one megabyte": {
			bytes:    uint64(math.Pow10(6)) - uint64(math.Pow10(3)),
			expected: "999.00 kB",
		},
		"one megabyte less than one gigabyte": {
			bytes:    uint64(math.Pow10(9)) - uint64(math.Pow10(6)),
			expected: "999.00 MB",
		},
		"one gigabyte less than one terabyte": {
			bytes:    uint64(math.Pow10(12)) - uint64(math.Pow10(9)),
			expected: "999.00 GB",
		},
		"one terabyte less than one petabyte": {
			bytes:    uint64(math.Pow10(15)) - uint64(math.Pow10(12)),
			expected: "999.00 TB",
		},
		"one petabyte less than one exabyte": {
			bytes:    uint64(math.Pow10(18)) - uint64(math.Pow10(15)),
			expected: "999.00 PB",
		},
		"one petabyte more than one exabyte": {
			bytes:    uint64(math.Pow10(18)) + uint64(math.Pow10(15)),
			expected: "1.00 EB",
		},
		"1.25 GB": {
			bytes:    uint64(math.Pow10(9) * 1.25),
			expected: "1.25 GB",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			actual := BytesToHumanReadableUnit(tt.bytes)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
