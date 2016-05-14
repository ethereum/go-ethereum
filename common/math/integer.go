package math

import gmath "math"

/*
 * NOTE: The following methods need to be optimised using either bit checking or asm
 */

// IsMinSafe returns whether the subtraction is safe and does not overflow.
func IsSubSafe(x, y uint64) bool {
	return x >= y
}

// IsAddSafe returns whether the addition is safe and does not overflow.
func IsAddSafe(x, y uint64) bool {
	return y <= gmath.MaxUint64-x
}

// IsAddSafe returns whether the multiplication is safe and does not overflow.
func IsMulSafe(x, y uint64) bool {
	return x == 0 || y == 0 || y <= gmath.MaxUint64/x
}
