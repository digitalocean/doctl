package template

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/text"
	"github.com/digitalocean/doctl/commands/charm/textbox"
	"github.com/dustin/go-humanize"
)

type FuncMap = template.FuncMap

// Funcs returns template helpers.
func Funcs(colors charm.ColorScheme) template.FuncMap {
	return template.FuncMap{
		"checkmark":    charm.Factory(text.Checkmark),
		"crossmark":    charm.Factory(text.Crossmark),
		"asterisk":     charm.Factory(text.Astreisk),
		"promptPrefix": charm.Factory(text.PromptPrefix),
		"pointerUp":    charm.Factory(text.PointerUp),
		"pointerRight": charm.Factory(text.PointerRight),
		"pointerDown":  charm.Factory(text.PointerDown),
		"pointerLeft":  charm.Factory(text.PointerLeft),
		"nl": func(n ...int) string {
			count := 1
			if len(n) > 0 {
				count = n[0]
			}
			return strings.Repeat("\n", count)
		},

		"newTextBox": textbox.New,

		"success":   text.Success.S,
		"warning":   text.Warning.S,
		"error":     text.Error.S,
		"highlight": text.Highlight.S,
		"muted":     text.Muted.S,

		"bold":      text.Bold.S,
		"underline": text.Underline.S,

		"lower":        strings.ToLower,
		"snakeToTitle": charm.SnakeToTitle,
		"join": func(sep string, pieces ...any) string {
			strs := make([]string, len(pieces))
			for i, p := range pieces {
				strs[i] = fmt.Sprint(p)
			}
			return strings.Join(strs, sep)
		},
		"duration": func(d time.Duration, precision ...string) string {
			var trunc time.Duration
			if len(precision) > 0 {
				switch strings.ToLower(precision[0]) {
				case "us":
					trunc = time.Microsecond
				case "ms":
					trunc = time.Millisecond
				case "s":
					trunc = time.Second
				case "m":
					trunc = time.Minute
				}
			}

			if trunc == 0 {
				switch {
				case d < time.Millisecond:
				case d < time.Second:
					trunc = time.Millisecond
				default:
					trunc = time.Second
				}
			}
			return d.Truncate(trunc).String()
		},
		"timeAgo": humanize.Time,
	}
}

// Render renders a template.
func Render(w io.Writer, content string, data any) error {
	tmpl := template.New("tmpl").Funcs(Funcs(charm.Colors))
	var err error
	tmpl, err = tmpl.Parse(content)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

// BufferedE executes a template and writes the result to the given writer once.
func BufferedE(w io.Writer, content string, data any) error {
	var buf bytes.Buffer
	err := Render(&buf, content, data)
	if err != nil {
		return err
	}
	_, err = buf.WriteTo(w)
	return err
}

// Buffered executes a template and writes the result to the given writer once. If an error occurs, it is written
// to the writer instead.
func Buffered(w io.Writer, content string, data any) {
	err := BufferedE(w, content, data)
	if err != nil {
		fmt.Fprintf(w, "%s", text.Error.S(err))
	}
}

// StringE executes a template and returns it as a string.
func StringE(content string, data any) (string, error) {
	var buf bytes.Buffer
	err := Render(&buf, content, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// String executes a template and returns it as a string. If an error occurs, the error text is returned instead.
func String(content string, data any) string {
	res, err := StringE(content, data)
	if err != nil {
		return err.Error()
	}
	return res
}

// PrintE executes a template and prints it directly to stdout.
func PrintE(content string, data any) error {
	return Render(os.Stdout, content, data)
}

// Print executes a template and prints it directly to stdout. If an error occurs, it is written to stderr as well.
func Print(content string, data any) {
	err := Render(os.Stdout, content, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", text.Error.S(err))
	}
}
