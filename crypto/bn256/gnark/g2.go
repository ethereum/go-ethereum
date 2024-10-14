package bn256

import (
	"github.com/consensys/gnark-crypto/ecc/bn254"
)

// G2 is the affine representation of a G2 group element.
//
// Since this code is used for precompiles, using Jacobian
// points are not beneficial because there are no intermediate
// points.
//
// Note: We also use this struct so that we can conform to the existing API
// that the precompiles want.
type G2 struct {
	inner bn254.G2Affine
}

// Add adds `a` and `b` together storing the result in `g`
func (g *G2) Add(a, b *G2) {
	g.inner.Add(&a.inner, &b.inner)
}

// Unmarshal deserializes `buf` into `g`
//
// Note: whether the serialization is of a compressed
// or an uncompressed point, is encoding in the bytes.
//
// For our purpose, the point will always be serialized as uncompressed
// ie 128 bytes.
//
// This method checks whether the point is on the curve and
// in the subgroup.
func (g *G2) Unmarshal(buf []byte) (int, error) {
	return g.inner.SetBytes(buf)
}

// Marshal serializes the point into a byte slice.
//
// Note: The point is serialized as uncompressed.
func (g *G2) Marshal() []byte {
	return g.inner.Marshal()
}
