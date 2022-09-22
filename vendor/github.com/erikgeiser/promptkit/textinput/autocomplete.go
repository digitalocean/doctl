package textinput

import (
	"sort"
	"strings"
)

// AutoCompleteFromSlice creates a case-insensitive auto-complete function from
// a slice of choices.
func AutoCompleteFromSlice(choices []string) func(string) []string {
	return autoCompleteFromSlice(choices, false)
}

// AutoCompleteFromSliceWithDefault creates a case-insensitive auto-complete
// function from a slice of choices with a default completion value that is
// inserted if the function is called on an empty input.
func AutoCompleteFromSliceWithDefault(
	choices []string, defaultValue string,
) func(string) []string {
	autoComplete := autoCompleteFromSlice(choices, false)

	return func(s string) []string {
		if s == "" {
			return []string{defaultValue}
		}

		return autoComplete(s)
	}
}

// CaseSensitiveAutoCompleteFromSlice creates a case-sensitive auto-complete
// function from a slice of choices.
func CaseSensitiveAutoCompleteFromSlice(choices []string) func(string) []string {
	return autoCompleteFromSlice(choices, true)
}

// CaseSensitiveAutoCompleteFromSliceWithDefault creates a case-sensitive
// auto-complete function from a slice of choices with a default completion
// value that is inserted if the function is called on an empty input.
func CaseSensitiveAutoCompleteFromSliceWithDefault(
	choices []string, defaultValue string,
) func(string) []string {
	autoComplete := autoCompleteFromSlice(choices, true)

	return func(s string) []string {
		if s == "" {
			return []string{defaultValue}
		}

		return autoComplete(s)
	}
}

func autoCompleteFromSlice(choices []string, caseSensitive bool) func(string) []string {
	return func(value string) []string {
		v := value
		if !caseSensitive {
			v = strings.ToLower(value)
		}

		var suggestions []string

		for _, choice := range choices {
			ch := choice
			if !caseSensitive {
				ch = strings.ToLower(choice)
			}

			if strings.HasPrefix(ch, v) {
				suggestions = append(suggestions, choice)
			}
		}

		return suggestions
	}
}

func commonPrefix(suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	} else if len(suggestions) == 1 {
		return suggestions[0]
	}

	// by sorting we ensure that the prefixes differ the most between the first
	// and the last element in the slice, therefore it is sufficient only to
	// determine the common prefix between these two
	sort.Strings(suggestions) // O(n*log(n))

	first := suggestions[0]
	last := suggestions[len(suggestions)-1]

	for i := 0; i < len(first); i++ {
		if last[i] != first[i] {
			return first[:i]
		}
	}

	return first
}
