package selection

import (
	"fmt"
)

// Choice represents a single choice. This type used as an input
// for the selection prompt, for filtering and as a result value.
type Choice[T any] struct {
	idx    int
	String string
	Value  T
}

func (c *Choice[T]) Index() int {
	return c.idx
}

// newChoice creates a new choice for a given input and chooses
// a suitable string representation. The index is left at 0 to
// be populated by the selection prompt later on.
func newChoice[T any](item T) *Choice[T] {
	choice := &Choice[T]{idx: 0, Value: item}

	switch i := any(item).(type) {
	case Choice[T]:
		choice.String = i.String
	case *Choice[T]:
		choice.String = i.String
	case string:
		choice.String = i
	case fmt.Stringer:
		choice.String = i.String()
	default:
		choice.String = fmt.Sprintf("%+v", i)
	}

	return choice
}

// asChoices converts a slice of anything to a slice of choices.
func asChoices[T any](choices []T) []*Choice[T] {
	choicesSlice := make([]*Choice[T], 0, len(choices))

	for _, choice := range choices {
		choicesSlice = append(choicesSlice, newChoice(choice))
	}

	return choicesSlice
}
