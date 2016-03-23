package match

import (
	"reflect"
	"testing"
)

func TestEveryOfIndex(t *testing.T) {
	for id, test := range []struct {
		matchers Matchers
		fixture  string
		index    int
		segments []int
	}{
		{
			Matchers{
				Any{},
				NewText("b"),
				NewText("c"),
			},
			"abc",
			-1,
			nil,
		},
		{
			Matchers{
				Any{},
				Prefix{"b"},
				Suffix{"c"},
			},
			"abc",
			1,
			[]int{2},
		},
	} {
		everyOf := EveryOf{test.matchers}
		index, segments := everyOf.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}
