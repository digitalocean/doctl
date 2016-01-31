package match

import (
	"reflect"
	"testing"
)

func TestAnyOfIndex(t *testing.T) {
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
			0,
			[]int{0, 1, 2, 3},
		},
		{
			Matchers{
				Prefix{"b"},
				Suffix{"c"},
			},
			"abc",
			0,
			[]int{3},
		},
		{
			Matchers{
				List{"[def]", false},
				List{"[abc]", false},
			},
			"abcdef",
			0,
			[]int{1},
		},
	} {
		everyOf := AnyOf{test.matchers}
		index, segments := everyOf.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}
