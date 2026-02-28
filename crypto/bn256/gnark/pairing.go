package bn256

import (
	"github.com/consensys/gnark-crypto/ecc/bn254"
)

// PairingCheck computes the following relation: ∏ᵢ e(Pᵢ, Qᵢ) =? 1
//
// To explain why gnark returns a (bool, error):
//
//   - If the function `e` does not return a result then internally
//     an error is returned.
//   - If `e` returns a result, then error will be nil,
//     but if this value is not `1` then the boolean value will be false
//
// We therefore check for an error, and return false if its non-nil and
// then return the value of the boolean if not.
func PairingCheck(a_ []*G1, b_ []*G2) bool {
	a := getInnerG1s(a_)
	b := getInnerG2s(b_)

	// Assume that len(a) == len(b)
	//
	// The pairing function will return
	// false, if this is not the case.
	size := len(a)

	// Check if input is empty -- gnark will
	// return false on an empty input, however
	// the ossified behavior is to return true
	// on an empty input, so we add this if statement.
	if size == 0 {
		return true
	}

	ok, err := bn254.PairingCheck(a, b)
	if err != nil {
		return false
	}
	return ok
}

// getInnerG1s gets the inner gnark G1 elements.
//
// These methods are used for two reasons:
//
//   - We use a new type `G1`, so we need to convert from
//     []*G1 to []*bn254.G1Affine
//   - The gnark API accepts slices of values and not slices of
//     pointers to values, so we need to return []bn254.G1Affine
//     instead of []*bn254.G1Affine.
func getInnerG1s(pointerSlice []*G1) []bn254.G1Affine {
	gnarkValues := make([]bn254.G1Affine, 0, len(pointerSlice))
	for _, ptr := range pointerSlice {
		if ptr != nil {
			gnarkValues = append(gnarkValues, ptr.inner)
		}
	}
	return gnarkValues
}

// getInnerG2s gets the inner gnark G2 elements.
//
// The rationale for this method is the same as `getInnerG1s`.
func getInnerG2s(pointerSlice []*G2) []bn254.G2Affine {
	gnarkValues := make([]bn254.G2Affine, 0, len(pointerSlice))
	for _, ptr := range pointerSlice {
		if ptr != nil {
			gnarkValues = append(gnarkValues, ptr.inner)
		}
	}
	return gnarkValues
}
