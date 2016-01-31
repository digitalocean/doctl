package match

import (
	"reflect"
	"testing"
)

func TestSuperIndex(t *testing.T) {
	for id, test := range []struct {
		fixture  string
		index    int
		segments []int
	}{
		{
			"abc",
			0,
			[]int{0, 1, 2, 3},
		},
		{
			"",
			0,
			[]int{0},
		},
	} {
		p := Super{}
		index, segments := p.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}

func BenchmarkIndexSuper(b *testing.B) {
	m := Super{}
	for i := 0; i < b.N; i++ {
		m.Index(bench_pattern)
	}
}
