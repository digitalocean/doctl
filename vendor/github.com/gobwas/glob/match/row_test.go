package match

import (
	"reflect"
	"testing"
)

func BenchmarkRowIndex(b *testing.B) {
	m := Row{
		Matchers: Matchers{
			NewText("abc"),
			NewText("def"),
			Single{},
		},
		RunesLength: 7,
	}
	for i := 0; i < b.N; i++ {
		m.Index("abcdefghijk")
	}
}

func TestRowIndex(t *testing.T) {
	for id, test := range []struct {
		matchers Matchers
		length   int
		fixture  string
		index    int
		segments []int
	}{
		{
			Matchers{
				NewText("abc"),
				NewText("def"),
				Single{},
			},
			7,
			"qweabcdefghij",
			3,
			[]int{7},
		},
		{
			Matchers{
				NewText("abc"),
				NewText("def"),
				Single{},
			},
			7,
			"abcd",
			-1,
			nil,
		},
	} {
		p := Row{
			Matchers:    test.matchers,
			RunesLength: test.length,
		}
		index, segments := p.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}
