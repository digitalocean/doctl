/*
Package promptkit is a collection of common command line prompts for interactive
programs. Each prompts comes with sensible defaults, re-mappable key bindings
and many opportunities for heavy customization.

The actual prompt components can be found in the sub directories.
*/
package promptkit

import (
	"bufio"
	"fmt"
	"strings"
	"text/template"

	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
)

// ErrAborted is returned when the prompt was aborted.
var ErrAborted = fmt.Errorf("prompt aborted")

// UtilFuncMap returns a template.FuncMap with handy utility functions for
// prompt templates.
//
//  * Repeat(string, int) string: Identical to strings.Repeat.
//  * Len(string): reflow/ansi.PrintableRuneWidth, Len works like len but is
//    aware of ansi codes and returns the length of the string as it appears
//    on the screen.
//  * Min(int, int) int: The minimum of two ints.
//  * Max(int, int) int: The maximum of two ints.
//  * Add(int, int) int: The sum of two ints.
//  * Sub(int, int) int: The difference of two ints.
//  * Mul(int, int) int: The product of two ints.
func UtilFuncMap() template.FuncMap {
	return template.FuncMap{
		"Repeat": strings.Repeat,
		"Len":    ansi.PrintableRuneWidth,
		"Min": func(a, b int) int {
			if a <= b {
				return a
			}

			return b
		},
		"Max": func(a, b int) int {
			if a >= b {
				return a
			}

			return b
		},
		"Add": func(a, b int) int { return a + b },
		"Sub": func(a, b int) int { return a - b },
		"Mul": func(a, b int) int { return a * b },
	}
}

// WrapMode decides in which way text is wrapped.
type WrapMode func(string, int) string

// WordWrap performs a word wrap on the input and forces a wrap at width if a
// word is still larger that width after soft wrapping. This is known to cause
// issues with coloring in some terminals depending on the prompt style.
func WordWrap(input string, width int) string {
	if width == 0 {
		return input
	}

	return wrap.String(wordwrap.String(input, width), width)
}

var _ WrapMode = WordWrap

// HardWrap performs a hard wrap at the given width.
func HardWrap(input string, width int) string {
	if width == 0 {
		return input
	}

	return wrap.String(input, width)
}

var _ WrapMode = HardWrap

// Truncate cuts the string after the given width.
func Truncate(input string, width int) string {
	if width == 0 {
		return input
	}

	var truncated strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		truncated.WriteString(truncate.String(scanner.Text(), uint(width)) + "\n")
	}

	return truncated.String()
}

var _ WrapMode = Truncate
