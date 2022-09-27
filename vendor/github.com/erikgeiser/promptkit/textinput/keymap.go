package textinput

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// NewDefaultKeyMap returns a KeyMap with sensible default key mappings that can
// also be used as a starting point for customization.
func NewDefaultKeyMap() *KeyMap {
	return &KeyMap{
		MoveBackward:           []string{"left", "ctrl+b"},
		MoveForward:            []string{"right", "ctrl+f"},
		JumpToBeginning:        []string{"home", "ctrl+a"},
		JumpToEnd:              []string{"end", "ctrl+e"},
		DeleteBeforeCursor:     []string{"backspace"},
		DeleteWordBeforeCursor: []string{"ctrl+w"},
		DeleteUnderCursor:      []string{"delete", "ctrl+d"},
		DeleteAllAfterCursor:   []string{"ctrl+k"},
		DeleteAllBeforeCursor:  []string{"ctrl+u"},
		AutoComplete:           []string{"tab"},
		Paste:                  []string{"ctrl+v"},
		Clear:                  []string{"esc"},
		Reset:                  []string{},
		Submit:                 []string{"enter"},
		Abort:                  []string{"ctrl+c"},
	}
}

var upsteamKeyMap = &KeyMap{
	MoveBackward:           []string{"left", "ctrl+b"},
	MoveForward:            []string{"right", "ctrl+f"},
	JumpToBeginning:        []string{"home", "ctrl+a"},
	JumpToEnd:              []string{"end", "ctrl+e"},
	DeleteBeforeCursor:     []string{"backspace"},
	DeleteWordBeforeCursor: []string{"ctrl+w"},
	DeleteUnderCursor:      []string{"delete", "ctrl+d"},
	DeleteAllAfterCursor:   []string{"ctrl+k"},
	DeleteAllBeforeCursor:  []string{"ctrl+u"},
	Paste:                  []string{"ctrl+v"},
}

// KeyMap defines the keys that trigger certain actions.
type KeyMap struct {
	MoveBackward           []string
	MoveForward            []string
	JumpToBeginning        []string
	JumpToEnd              []string
	DeleteBeforeCursor     []string
	DeleteWordBeforeCursor []string
	DeleteUnderCursor      []string
	DeleteAllAfterCursor   []string
	DeleteAllBeforeCursor  []string
	AutoComplete           []string
	Paste                  []string
	Clear                  []string
	Reset                  []string
	Submit                 []string
	Abort                  []string
}

func keyMatches(key tea.KeyMsg, mapping []string) bool {
	for _, m := range mapping {
		if m == key.String() {
			return true
		}
	}

	return false
}

func keyMatchesUpstreamKeyMap(key tea.KeyMsg) bool {
	return keyMatches(key, allKeys(upsteamKeyMap))
}

// validateKeyMap returns true if the given key map contains at
// least the bare minimum set of key bindings for the functional
// prompt and false otherwise.
func validateKeyMap(km *KeyMap) error {
	if len(km.Submit) == 0 {
		return fmt.Errorf("no submit key")
	}

	if len(km.Abort) == 0 {
		return fmt.Errorf("no abort key")
	}

	return nil
}

func allKeys(km *KeyMap) (keys []string) {
	keys = append(keys, km.MoveBackward...)
	keys = append(keys, km.MoveForward...)
	keys = append(keys, km.JumpToBeginning...)
	keys = append(keys, km.JumpToEnd...)
	keys = append(keys, km.DeleteBeforeCursor...)
	keys = append(keys, km.DeleteUnderCursor...)
	keys = append(keys, km.DeleteAllAfterCursor...)
	keys = append(keys, km.DeleteAllBeforeCursor...)
	keys = append(keys, km.AutoComplete...)
	keys = append(keys, km.Paste...)
	keys = append(keys, km.Clear...)
	keys = append(keys, km.Reset...)
	keys = append(keys, km.Submit...)
	keys = append(keys, km.Abort...)

	return keys
}
