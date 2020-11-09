package go4k_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/vsariola/sointu/go4k"
)

func TestFindSuperIntArray(t *testing.T) {
	var tests = []struct {
		input       [][]int
		wantSuper   []int
		wantIndices []int
	}{
		{[][]int{}, []int{}, []int{}},
		{[][]int{nil, nil}, []int{}, []int{0, 0}},
		{[][]int{{3, 4, 5}, {1, 2, 3}}, []int{1, 2, 3, 4, 5}, []int{2, 0}},
		{[][]int{{3, 4, 5}, {1, 2, 3}, nil}, []int{1, 2, 3, 4, 5}, []int{2, 0, 0}},
		{[][]int{{3, 4, 5}, {1, 2, 3}, {}}, []int{1, 2, 3, 4, 5}, []int{2, 0, 0}},
		{[][]int{{3, 4, 5}, {1, 2, 3}, {2, 3}}, []int{1, 2, 3, 4, 5}, []int{2, 0, 1}},
		{[][]int{{1, 2, 3, 4, 5}, {1, 2, 3}}, []int{1, 2, 3, 4, 5}, []int{0, 0}},
		{[][]int{{1, 2, 3, 4, 5}, {2, 3}}, []int{1, 2, 3, 4, 5}, []int{0, 1}},
		{[][]int{{1, 2, 3, 4, 5}, {2, 3}, {5, 6, 7}}, []int{1, 2, 3, 4, 5, 6, 7}, []int{0, 1, 4}},
		{[][]int{{1, 2, 3, 4}, {3, 4, 1}, {2, 3, 4, 5}}, []int{3, 4, 1, 2, 3, 4, 5}, []int{2, 0, 3}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("TestFindSuperIntArray %d", i), func(t *testing.T) {
			super, indices := go4k.FindSuperIntArray(tt.input)
			if !reflect.DeepEqual(super, tt.wantSuper) || !reflect.DeepEqual(indices, tt.wantIndices) {
				t.Errorf("FindSuperIntArray(%v) got (%v,%v), want (%v,%v)", tt.input, super, indices, tt.wantSuper, tt.wantIndices)
			}
		})
	}
}
