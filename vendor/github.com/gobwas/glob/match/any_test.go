package match

import (
	"reflect"
	"testing"
)

func TestAnyIndex(t *testing.T) {
	for id, test := range []struct {
		sep      string
		fixture  string
		index    int
		segments []int
	}{
		{
			".",
			"abc",
			0,
			[]int{0, 1, 2, 3},
		},
		{
			".",
			"abc.def",
			0,
			[]int{0, 1, 2, 3},
		},
	} {
		p := Any{test.sep}
		index, segments := p.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}

func BenchmarkIndexAny(b *testing.B) {
	p := Any{bench_separators}
	for i := 0; i < b.N; i++ {
		p.Index(bench_pattern)
	}
}
