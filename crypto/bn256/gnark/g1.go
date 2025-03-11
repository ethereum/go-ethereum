package bn256

import (
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
// Note: whether the deserialization is of a compressed
// or an uncompressed point, is encoded in the bytes.
//
// For our purpose, the point will always be serialized
// as uncompressed, ie 64 bytes.
//
// This method also checks whether the point is on the
// curve and in the prime order subgroup.
func (g *G1) Unmarshal(buf []byte) (int, error) {
	return g.inner.SetBytes(buf)
}

// Marshal serializes the point into a byte slice.
//
// Note: The point is serialized as uncompressed.
func (p *G1) Marshal() []byte {
	return p.inner.Marshal()
}
