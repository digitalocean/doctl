package match

import (
	"reflect"
	"testing"
)

func TestMinIndex(t *testing.T) {
	for id, test := range []struct {
		limit    int
		fixture  string
		index    int
		segments []int
	}{
		{
			1,
			"abc",
			0,
			[]int{1, 2, 3},
		},
		{
			3,
			"abcd",
			0,
			[]int{3, 4},
		},
	} {
		p := Min{test.limit}
		index, segments := p.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}

func BenchmarkIndexMin(b *testing.B) {
	m := Min{10}
	for i := 0; i < b.N; i++ {
		m.Index(bench_pattern)
	}
}
