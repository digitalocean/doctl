package confirmation

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// NewDefaultKeyMap returns a KeyMap with sensible default key mappings that can
// also be used as a starting point for customization.
func NewDefaultKeyMap() *KeyMap {
	return &KeyMap{
		Yes:       []string{"y", "Y"},
		No:        []string{"n", "N"},
		SelectYes: []string{"left"},
		SelectNo:  []string{"right"},
		Toggle:    []string{"tab"},
		Submit:    []string{"enter"},
		Abort:     []string{"ctrl+c"},
	}
}

// KeyMap defines the keys that trigger certain actions.
type KeyMap struct {
	Yes       []string
	No        []string
	SelectYes []string
	SelectNo  []string
	Toggle    []string
	Submit    []string
	Abort     []string
}

func keyMatches(key tea.KeyMsg, mapping []string) bool {
	for _, m := range mapping {
		if m == key.String() {
			return true
		}
	}

	return false
}

// validateKeyMap returns true if the given key map contains at
// least the bare minimum set of key bindings for the functional
// prompt and false otherwise.
func validateKeyMap(km *KeyMap) error {
	if len(km.Yes) == 0 && len(km.No) == 0 && len(km.Submit) == 0 {
		return fmt.Errorf("no submit key")
	}

	if !(len(km.Yes) > 0 && len(km.No) > 0) &&
		len(km.Toggle) == 0 &&
		!(len(km.SelectYes) > 0 && len(km.SelectNo) > 0) {
		return fmt.Errorf("missing keys to select a value")
	}

	return nil
}
