/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Harbor selects task files with the Python pathspec library, and a task's
// digest depends on exactly which files are selected. This file ports the two
// pathspec flavours Harbor uses so that doctl selects the same files:
//
//   - harborIgnoreBasic is pathspec's "gitignore" PathSpec, used for the
//     task's root .gitignore. The last matching pattern wins.
//   - harborIgnoreGit is pathspec's GitIgnoreSpec, used for .harborignore.
//     A file-level match outranks a directory-level one, as in Git.
type harborIgnoreFlavour int

const (
	harborIgnoreBasic harborIgnoreFlavour = iota
	harborIgnoreGit
)

const harborDirMark = "ps_d"

type harborIgnorePattern struct {
	re *regexp.Regexp
	// exclude is pathspec's "include": true for a normal pattern, false for
	// a negated (!) one.
	exclude bool
	dirMark int
}

type harborIgnoreSpec struct {
	flavour  harborIgnoreFlavour
	patterns []harborIgnorePattern
}

type harborRangeError struct{ pattern string }

func (e harborRangeError) Error() string {
	return fmt.Sprintf("invalid range notation in pattern %q", e.pattern)
}

func newHarborIgnoreSpec(flavour harborIgnoreFlavour, lines []string) (*harborIgnoreSpec, error) {
	spec := &harborIgnoreSpec{flavour: flavour}
	for _, line := range lines {
		if line == "" {
			continue
		}
		expr, exclude, ok, err := harborPatternToRegex(flavour, line)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("invalid ignore pattern %q: %w", line, err)
		}
		spec.patterns = append(spec.patterns, harborIgnorePattern{
			re:      re,
			exclude: exclude,
			dirMark: re.SubexpIndex(harborDirMark),
		})
	}
	return spec, nil
}

// check reports whether path is excluded, and whether any pattern matched at
// all (a negated match re-includes the path).
func (s *harborIgnoreSpec) check(path string) (excluded, matched bool) {
	if s.flavour == harborIgnoreBasic {
		for i := len(s.patterns) - 1; i >= 0; i-- {
			if s.patterns[i].re.MatchString(path) {
				return s.patterns[i].exclude, true
			}
		}
		return false, false
	}

	priority := 0
	for i := len(s.patterns) - 1; i >= 0; i-- {
		p := s.patterns[i]
		loc := p.re.FindStringSubmatchIndex(path)
		if loc == nil {
			continue
		}
		prio := 2
		if p.dirMark > 0 && loc[2*p.dirMark] >= 0 && loc[2*p.dirMark+1] > loc[2*p.dirMark] {
			prio = 1
		}
		if prio > priority {
			excluded, matched, priority = p.exclude, true, prio
		}
		if prio == 2 {
			break
		}
	}
	return excluded, matched
}

func (s *harborIgnoreSpec) excludes(path string) bool {
	excluded, _ := s.check(path)
	return excluded
}

// harborPatternToRegex ports pattern_to_regex from pathspec's
// GitIgnoreBasicPattern and GitIgnoreSpecPattern. ok is false for blank lines,
// comments, and patterns pathspec discards.
func harborPatternToRegex(flavour harborIgnoreFlavour, pattern string) (expr string, exclude, ok bool, err error) {
	original := pattern
	if !strings.HasSuffix(pattern, `\ `) {
		pattern = strings.TrimRightFunc(pattern, unicode.IsSpace)
	}
	if pattern == "" || strings.HasPrefix(pattern, "#") {
		return "", false, false, nil
	}
	if flavour == harborIgnoreGit && pattern == "/" {
		return "", false, false, nil
	}

	exclude = true
	if strings.HasPrefix(pattern, "!") {
		exclude = false
		pattern = pattern[1:]
	}

	segs := strings.Split(pattern, "/")
	isDir := segs[len(segs)-1] == ""
	if flavour == harborIgnoreBasic && pattern == "/" {
		isDir = false
	}

	segs, override, err := harborNormalizeSegments(flavour, isDir, segs)
	if err != nil {
		return "", false, false, fmt.Errorf("invalid ignore pattern %q: %w", original, err)
	}
	if override != "" {
		return override, exclude, true, nil
	}

	expr, err = harborTranslateSegments(flavour, isDir, segs)
	if err != nil {
		if _, discard := err.(harborRangeError); discard {
			return "", false, false, nil
		}
		return "", false, false, fmt.Errorf("invalid ignore pattern %q: %w", original, err)
	}
	return expr, exclude, true, nil
}

func harborNormalizeSegments(flavour harborIgnoreFlavour, isDir bool, segs []string) ([]string, string, error) {
	switch {
	case segs[0] == "":
		segs = segs[1:]
	case len(segs) == 1 || (len(segs) == 2 && segs[1] == ""):
		if segs[0] != "**" {
			segs = append([]string{"**"}, segs...)
		}
	}
	if len(segs) == 0 {
		return nil, "", fmt.Errorf("pattern normalized to nothing")
	}
	if segs[len(segs)-1] == "" {
		segs[len(segs)-1] = "**"
	}
	for i := len(segs) - 1; i > 0; i-- {
		if segs[i-1] == "**" && segs[i] == "**" {
			segs = append(segs[:i], segs[i+1:]...)
		}
	}

	dirMark := "(?P<" + harborDirMark + ">/)"
	switch {
	case len(segs) == 1 && segs[0] == "**":
		if isDir {
			if flavour == harborIgnoreGit {
				return nil, dirMark, nil
			}
			return nil, "/", nil
		}
		return nil, ".", nil
	case len(segs) == 2 && segs[0] == "**" && segs[1] == "*":
		return nil, ".", nil
	case len(segs) == 3 && segs[0] == "**" && segs[1] == "*" && segs[2] == "**":
		if isDir && flavour == harborIgnoreGit {
			return nil, dirMark, nil
		}
		return nil, "/", nil
	}
	return segs, "", nil
}

func harborTranslateSegments(flavour harborIgnoreFlavour, isDir bool, segs []string) (string, error) {
	var out strings.Builder
	needSlash := false
	end := len(segs) - 1
	for i, seg := range segs {
		if seg == "**" {
			switch {
			case i == 0:
				out.WriteString("^(?:.+/)?")
			case i < end:
				out.WriteString("(?:/.+)?")
				needSlash = true
			case isDir && flavour == harborIgnoreGit:
				out.WriteString("(?P<" + harborDirMark + ">/)")
			default:
				out.WriteString("/")
			}
			continue
		}

		if i == 0 {
			out.WriteString("^")
		}
		if needSlash {
			out.WriteString("/")
		}
		if seg == "*" {
			out.WriteString("[^/]+")
		} else {
			glob, err := harborTranslateGlob(seg, flavour == harborIgnoreGit)
			if err != nil {
				return "", err
			}
			out.WriteString(glob)
		}
		if i == end {
			switch {
			case flavour == harborIgnoreGit:
				out.WriteString("(?:(?P<" + harborDirMark + ">/)|$)")
			case seg == "*":
				out.WriteString("/?$")
			default:
				out.WriteString("(?:/|$)")
			}
		}
		needSlash = true
	}
	return out.String(), nil
}

// harborTranslateGlob ports _translate_segment_glob. With rangeErrors set, an
// unterminated bracket expression is an error (the Git flavour then discards
// the pattern); otherwise the bracket is taken literally.
func harborTranslateGlob(pattern string, rangeErrors bool) (string, error) {
	chars := []rune(pattern)
	var out strings.Builder
	escape := false
	for i := 0; i < len(chars); {
		c := chars[i]
		i++
		switch {
		case escape:
			escape = false
			out.WriteString(regexp.QuoteMeta(string(c)))
		case c == '\\':
			escape = true
		case c == '*':
			out.WriteString("[^/]*")
		case c == '?':
			out.WriteString("[^/]")
		case c == '[':
			j := i
			if j < len(chars) && (chars[j] == '!' || chars[j] == '^') {
				j++
			}
			if j < len(chars) && chars[j] == ']' {
				j++
			}
			for j < len(chars) && chars[j] != ']' {
				j++
			}
			if j >= len(chars) {
				if rangeErrors {
					return "", harborRangeError{pattern: pattern}
				}
				out.WriteString(`\[`)
				continue
			}
			j++
			out.WriteString("[")
			if chars[i] == '!' || chars[i] == '^' {
				out.WriteString("^")
				i++
			}
			out.WriteString(strings.ReplaceAll(string(chars[i:j]), `\`, `\\`))
			i = j
		default:
			out.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	if escape {
		return "", fmt.Errorf("escape character found with no next character to escape")
	}
	return out.String(), nil
}

// harborSplitLines ports Python's str.splitlines, which .gitignore and
// .harborignore are read with.
func harborSplitLines(s string) []string {
	var lines []string
	start := 0
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '\r':
			lines = append(lines, string(runes[start:i]))
			if i+1 < len(runes) && runes[i+1] == '\n' {
				i++
			}
			start = i + 1
		case '\n', '\v', '\f', '\x1c', '\x1d', '\x1e', '\u0085', '\u2028', '\u2029':
			lines = append(lines, string(runes[start:i]))
			start = i + 1
		}
	}
	if start < len(runes) {
		lines = append(lines, string(runes[start:]))
	}
	return lines
}
