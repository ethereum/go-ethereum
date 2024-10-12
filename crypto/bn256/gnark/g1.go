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

// Add adds `a` and `b` together storing the result in `g`
func (g *G1) Add(a, b *G1) {
	// TODO(Decision to be made): There are three ways to
	// TODO do this addition. Each with different performance
	// TODO: characteristics.
	//
	// Option 1: This just calls a method in gnark
	// g.inner.Add(&a.inner, &b.inner)

	// Option 2: This calls multiple methods in gnark
	// but is faster.
	//
	// var res bn254.G1Jac
	// res.FromAffine(&a.inner)
	// res.AddMixed(&b.inner)
	// g.inner.FromJacobian(&res)

	// Option 3: This calls a method that I created that
	// we can upstream to gnark.
	// This should be the fastest, I can write the same for G2
	g.addAffine(a, b)
}

// ScalarMult computes the scalar multiplication between `a` and
// `scalar` storing the result in `g`
func (g *G1) ScalarMult(a *G1, scalar *big.Int) {
	g.inner.ScalarMultiplication(&a.inner, scalar)
}

// Double adds `a` to itself, storing the result in `g`
func (g *G1) Double(a *G1) {
	g.inner.Double(&a.inner)
}

// Unmarshal deserializes `buf` into `g`
//
// Note: whether the serialization is of a compressed
// or an uncompressed point, is encoding in the bytes.
//
// For our purpose, the point will always be serialized as uncompressed
// ie 64 bytes.
//
// This method checks whether the point is on the curve and
// in the subgroup.
func (g *G1) Unmarshal(buf []byte) (int, error) {
	return g.inner.SetBytes(buf)
}

// Marshal serializes the point into a byte slice.
//
// Note: The point is serialized as uncompressed.
func (p *G1) Marshal() []byte {
	return p.inner.Marshal()
}
