package math

import (
	gmath "math"
	"testing"
)

type operation byte

const (
	sub operation = iota
	add
	mul
)

func TestIsAddSafe(t *testing.T) {
	for i, test := range []struct {
		x    uint64
		y    uint64
		safe bool
		op   operation
	}{
		// add operations
		{gmath.MaxUint64, 1, false, add},
		{gmath.MaxUint64 - 1, 1, true, add},

		// sub operations
		{0, 1, false, sub},
		{0, 0, true, sub},

		// mul operations
		{10, 10, true, mul},
		{gmath.MaxUint64, 2, false, mul},
		{gmath.MaxUint64, 1, true, mul},
	} {
		var isSafe bool
		switch test.op {
		case sub:
			isSafe = IsSubSafe(test.x, test.y)
		case add:
			isSafe = IsAddSafe(test.x, test.y)
		case mul:
			isSafe = IsMulSafe(test.x, test.y)
		}

		if test.safe != isSafe {
			t.Errorf("%d failed. Expected test to be %v, got %v", i, test.safe, isSafe)
		}
	}
}
