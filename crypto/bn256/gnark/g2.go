package bn256

import (
	"errors"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ethereum/go-ethereum/common/bitutil"
)

// G2 is the affine representation of a G2 group element.
//
// Since this code is used for precompiles, using Jacobian
// points are not beneficial because there are no intermediate
// points and G2 in particular is only used for the pairing input.
//
// Note: We also use this struct so that we can conform to the existing API
// that the precompiles want.
type G2 struct {
	inner bn254.G2Affine
}

// Unmarshal deserializes `buf` into `g`
//
// The input is expected to be in the EVM format:
// 128 bytes: [32-byte x.1][32-byte x.0][32-byte y.1][32-byte y.0]
// where each value is a big-endian integer.
//
// This method also checks whether the point is on the
// curve and in the prime order subgroup.
func (g *G2) Unmarshal(buf []byte) (int, error) {
	if len(buf) < 128 {
		return 0, errors.New("invalid G2 point size")
	}

	if !bitutil.TestBytes(buf[:128]) {
		// point at infinity
		g.inner.X.A0.SetZero()
		g.inner.X.A1.SetZero()
		g.inner.Y.A0.SetZero()
		g.inner.Y.A1.SetZero()
		return 128, nil
	}
	if err := g.inner.X.A1.SetBytesCanonical(buf[0:32]); err != nil {
		return 0, err
	}
	if err := g.inner.X.A0.SetBytesCanonical(buf[32:64]); err != nil {
		return 0, err
	}
	if err := g.inner.Y.A1.SetBytesCanonical(buf[64:96]); err != nil {
		return 0, err
	}
	if err := g.inner.Y.A0.SetBytesCanonical(buf[96:128]); err != nil {
		return 0, err
	}

	if !g.inner.IsOnCurve() {
		return 0, errors.New("point is not on curve")
	}
	if !g.inner.IsInSubGroup() {
		return 0, errors.New("point is not in correct subgroup")
	}
	return 128, nil
}

// Marshal serializes the point into a byte slice.
//
// The output is in EVM format: 128 bytes total.
// [32-byte x.1][32-byte x.0][32-byte y.1][32-byte y.0]
// where each value is a big-endian integer.
func (g *G2) Marshal() []byte {
	output := make([]byte, 128)

	xA1Bytes := g.inner.X.A1.Bytes()
	copy(output[:32], xA1Bytes[:])

	xA0Bytes := g.inner.X.A0.Bytes()
	copy(output[32:64], xA0Bytes[:])

	yA1Bytes := g.inner.Y.A1.Bytes()
	copy(output[64:96], yA1Bytes[:])

	yA0Bytes := g.inner.Y.A0.Bytes()
	copy(output[96:128], yA0Bytes[:])

	return output
}
