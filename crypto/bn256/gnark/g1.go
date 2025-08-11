package bn256

import (
	"errors"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254"
)

// G1 is the affine representation of a G1 group element.
//
// Since this code is used for precompiles, using Jacobian
// points are not beneficial because there are no intermediate
// points to allow us to save on inversions.
//
// Note: We also use this struct so that we can conform to the existing API
// that the precompiles want.
type G1 struct {
	inner bn254.G1Affine
}

// Add adds `a` and `b` together, storing the result in `g`
func (g *G1) Add(a, b *G1) {
	g.inner.Add(&a.inner, &b.inner)
}

// ScalarMult computes the scalar multiplication between `a` and
// `scalar`, storing the result in `g`
func (g *G1) ScalarMult(a *G1, scalar *big.Int) {
	g.inner.ScalarMultiplication(&a.inner, scalar)
}

// Unmarshal deserializes `buf` into `g`
//
// The input is expected to be in the EVM format:
// 64 bytes: [32-byte x coordinate][32-byte y coordinate]
// where each coordinate is in big-endian format.
//
// This method also checks whether the point is on the
// curve and in the prime order subgroup.
func (g *G1) Unmarshal(buf []byte) (int, error) {
	if len(buf) < 64 {
		return 0, errors.New("invalid G1 point size")
	}

	if allZeroes(buf[:64]) {
		// point at infinity
		g.inner.X.SetZero()
		g.inner.Y.SetZero()
		return 64, nil
	}

	if err := g.inner.X.SetBytesCanonical(buf[:32]); err != nil {
		return 0, err
	}
	if err := g.inner.Y.SetBytesCanonical(buf[32:64]); err != nil {
		return 0, err
	}

	if !g.inner.IsOnCurve() {
		return 0, errors.New("point is not on curve")
	}
	if !g.inner.IsInSubGroup() {
		return 0, errors.New("point is not in correct subgroup")
	}
	return 64, nil
}

// Marshal serializes the point into a byte slice.
//
// The output is in EVM format: 64 bytes total.
// [32-byte x coordinate][32-byte y coordinate]
// where each coordinate is a big-endian integer padded to 32 bytes.
func (p *G1) Marshal() []byte {
	output := make([]byte, 64)

	xBytes := p.inner.X.Bytes()
	copy(output[:32], xBytes[:])

	yBytes := p.inner.Y.Bytes()
	copy(output[32:64], yBytes[:])

	return output
}

func allZeroes(buf []byte) bool {
	for i := range buf {
		if buf[i] != 0 {
			return false
		}
	}
	return true
}
