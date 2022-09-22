package charm

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
)

func Margin(i ...int) Style {
	return NewStyle(lipgloss.NewStyle().Margin(i...))
}

func Indent(level uint) io.Writer {
	return indent.NewWriterPipe(os.Stdout, level, nil)
}

func IndentWriter(w io.Writer, level uint) io.Writer {
	return indent.NewWriterPipe(w, level, nil)
}

func IndentString(level uint, str string) string {
	return indent.String(str, level)
}

func Factory[T any](x T) func() T {
	return func() T {
		return x
	}
}

func SnakeToTitle(s any) string {
	return strings.Title(strings.ReplaceAll(strings.ToLower(fmt.Sprint(s)), "_", " "))
}
