package commands

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func Test_checkErr(t *testing.T) {
	defer func(a func()) { errAction = a }(errAction)
	defer func(a io.Writer) { color.Output = a }(color.Output)

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	color.Output = w

	errAction = func() {
	}

	e := errors.New("an error")
	checkErr(e)
	err := w.Flush()
	assert.NoError(t, err)

	// test for those color codes
	expected := "\n\x1b[31mError\x1b[0m: an error\n"

	assert.Equal(t, expected, b.String())
}
