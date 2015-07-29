package doit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_argSlicer(t *testing.T) {
	cases := []struct {
		input    []string
		expected [][]string
	}{
		{
			input:    []string{"foo=1", "bar=2"},
			expected: [][]string{{"foo", "1"}, {"bar", "2"}},
		},
	}

	for _, c := range cases {
		got := argSlicer(c.input)
		assert.Equal(t, c.expected, got)
	}
}
