package match

import (
	"reflect"
	"testing"
)

func TestListIndex(t *testing.T) {
	for id, test := range []struct {
		list     string
		not      bool
		fixture  string
		index    int
		segments []int
	}{
		{
			"ab",
			false,
			"abc",
			0,
			[]int{1},
		},
		{
			"ab",
			true,
			"fffabfff",
			0,
			[]int{1},
		},
	} {
		p := List{test.list, test.not}
		index, segments := p.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}

func BenchmarkIndexList(b *testing.B) {
	m := List{"def", false}
	for i := 0; i < b.N; i++ {
		m.Index(bench_pattern)
	}
}
