package match

import (
	"testing"
)

func TestBTree(t *testing.T) {
	for id, test := range []struct {
		tree BTree
		str  string
		exp  bool
	}{
		{
			NewBTree(NewText("abc"), Super{}, Super{}),
			"abc",
			true,
		},
		{
			NewBTree(NewText("a"), Single{}, Single{}),
			"aaa",
			true,
		},
		{
			NewBTree(NewText("b"), Single{}, nil),
			"bbb",
			false,
		},
		{
			NewBTree(
				NewText("c"),
				NewBTree(
					Single{},
					Super{},
					nil,
				),
				nil,
			),
			"abc",
			true,
		},
	} {
		act := test.tree.Match(test.str)
		if act != test.exp {
			t.Errorf("#%d match %q error: act: %t; exp: %t", id, test.str, act, test.exp)
			continue
		}
	}
}
