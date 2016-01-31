package match

import (
	"reflect"
	"testing"
)

func TestNothingIndex(t *testing.T) {
	for id, test := range []struct {
		fixture  string
		index    int
		segments []int
	}{
		{
			"abc",
			0,
			[]int{0},
		},
		{
			"",
			0,
			[]int{0},
		},
	} {
		p := Nothing{}
		index, segments := p.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}

func BenchmarkIndexNothing(b *testing.B) {
	m := Max{10}
	for i := 0; i < b.N; i++ {
		m.Index(bench_pattern)
	}
}
