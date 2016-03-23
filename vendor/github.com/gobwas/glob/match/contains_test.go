package match

import (
	"reflect"
	"testing"
)

func TestContainsIndex(t *testing.T) {
	for id, test := range []struct {
		prefix   string
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
			[]int{2, 3},
		},
		{
			"ab",
			false,
			"fffabfff",
			0,
			[]int{5, 6, 7, 8},
		},
		{
			"ab",
			true,
			"abc",
			0,
			[]int{0},
		},
		{
			"ab",
			true,
			"fffabfff",
			0,
			[]int{0, 1, 2, 3},
		},
	} {
		p := Contains{test.prefix, test.not}
		index, segments := p.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}

func BenchmarkIndexContains(b *testing.B) {
	m := Contains{bench_separators, true}
	for i := 0; i < b.N; i++ {
		m.Index(bench_pattern)
	}
}
