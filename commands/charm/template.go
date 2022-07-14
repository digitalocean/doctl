package charm

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"
)

// TemplateFuncs returns template helpers.
func TemplateFuncs(colors ColorScheme) template.FuncMap {
	return template.FuncMap{
		"checkmark":  factory(Checkmark),
		"crossmark":  factory(Crossmark),
		"newTextBox": NewTextBox,

		"success":   TextSuccess.S,
		"warning":   TextWarning.S,
		"error":     TextError.S,
		"highlight": TextHighlight.S,
		"bold":      TextBold.S,
		"underline": TextUnderline.S,
		"nl":        factory("\n"),

		"join": func(sep string, pieces ...any) string {
			strs := make([]string, len(pieces))
			for i, p := range pieces {
				strs[i] = fmt.Sprint(p)
			}
			return strings.Join(strs, sep)
		},
		"duration": func(d time.Duration, precision ...string) string {
			trunc := time.Second
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
			return d.Truncate(trunc).String()
		},
	}
}

// Template executes a template.
func Template(w io.Writer, content string, data any) error {
	tmpl := template.New("tmpl").Funcs(TemplateFuncs(Colors))
	var err error
	tmpl, err = tmpl.Parse(content)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

// TemplateBuffered executes a template and writes the result to the given writer once.
func TemplateBuffered(w io.Writer, content string, data any) error {
	var buf bytes.Buffer
	err := Template(&buf, content, data)
	if err != nil {
		return err
	}
	_, err = buf.WriteTo(w)
	return err
}

// TemplateString executes a template and returns it as a string.
func TemplateString(content string, data any) (string, error) {
	var buf bytes.Buffer
	err := Template(&buf, content, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// TemplatePrint executes a template and prints it directly to stdout.
func TemplatePrint(content string, data any) error {
	return Template(os.Stdout, content, data)
}
