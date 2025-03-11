package bn256

import (
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254"
)

// GT is the affine representation of a GT field element.
//
// Note: GT is not explicitly used in mainline code.
// It is needed for fuzzing.
type GT struct {
	inner bn254.GT
}

// Pair compute the optimal Ate pairing between a G1 and
// G2 element.
//
// Note: This method is not explicitly used in mainline code.
// It is needed for fuzzing. It should also be noted,
// that the output of this function may not match other
func Pair(a_ *G1, b_ *G2) *GT {
	a := a_.inner
	b := b_.inner

	pairingOutput, err := bn254.Pair([]bn254.G1Affine{a}, []bn254.G2Affine{b})

	if err != nil {
		// Since this method is only called during fuzzing, it is okay to panic here.
		// We do not return an error to match the interface of the other bn256 libraries.
		panic(fmt.Sprintf("gnark/bn254 encountered error: %v", err))
	}

	return &GT{
		inner: pairingOutput,
	}
}

// Unmarshal deserializes `buf` into `g`
//
// Note: This method is not explicitly used in mainline code.
// It is needed for fuzzing.
func (g *GT) Unmarshal(buf []byte) error {
	return g.inner.SetBytes(buf)
}

// Marshal serializes the point into a byte slice.
//
// Note: This method is not explicitly used in mainline code.
// It is needed for fuzzing.
func (g *GT) Marshal() []byte {
	bytes := g.inner.Bytes()
	return bytes[:]
}

// Exp raises `base` to the power of `exponent`
//
// Note: This method is not explicitly used in mainline code.
// It is needed for fuzzing.
func (g *GT) Exp(base GT, exponent *big.Int) *GT {
	g.inner.Exp(base.inner, exponent)
	return g
}
