package match

import (
	"reflect"
	"testing"
)

const bench_separators = "."
const bench_pattern = "abcdefghijklmnopqrstuvwxyz0123456789"

func TestMergeSegments(t *testing.T) {
	for id, test := range []struct {
		segments [][]int
		exp      []int
	}{
		{
			[][]int{
				[]int{0, 6, 7},
				[]int{0, 1, 3},
				[]int{2, 4},
			},
			[]int{0, 1, 2, 3, 4, 6, 7},
		},
		{
			[][]int{
				[]int{0, 1, 3, 6, 7},
				[]int{0, 1, 3},
				[]int{2, 4},
				[]int{1},
			},
			[]int{0, 1, 2, 3, 4, 6, 7},
		},
	} {
		act := mergeSegments(test.segments)
		if !reflect.DeepEqual(act, test.exp) {
			t.Errorf("#%d merge sort segments unexpected:\nact: %v\nexp:%v", id, act, test.exp)
			continue
		}
	}
}
