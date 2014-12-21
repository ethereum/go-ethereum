package whisper

import "testing"

func TestSorting(t *testing.T) {
	m := map[int32]Hash{
		1: HS("1"),
		3: HS("3"),
		2: HS("2"),
		5: HS("5"),
	}
	exp := []int32{1, 2, 3, 5}
	res := sortKeys(m)
	for i, k := range res {
		if k != exp[i] {
			t.Error(k, "failed. Expected", exp[i])
		}
	}
}
