package bn256

import (
	"github.com/consensys/gnark-crypto/ecc/bn254/fp"
)

// This is just the addition formula
// but given we know that we do not need Jacobian
// coordinates, we use the naive implementation.
//
// Ideally, we push this into gnark
func (g *G1) addAffine(a_, b_ *G1) {

	// Get the gnark specific points
	var a = a_.inner
	var b = b_.inner

	// If a is 0, then return b
	if a.IsInfinity() {
		g.inner.Set(&b)
		return
	}

	// If b is 0, then return a
	if b.IsInfinity() {
		g.inner.Set(&a)
		return
	}

	// If a == -b, then return 0
	g.inner.Neg(&b)
	if a.Equal(&g.inner) {
		g.inner.X.SetZero()
		g.inner.Y.SetZero()
		return
	}

	// Compute lambda based on whether we
	// are doing a point addition or a point doubling
	//
	// Check if points are equal
	var pointsAreEqual = a.Equal(&b)

	var denominator fp.Element
	var lambda fp.Element

	// If a == b, then we need to compute lambda for double
	// else we need to compute lambda for addition
	if pointsAreEqual {
		// Compute numerator
		lambda.Square(&a.X)
		fp.MulBy3(&lambda)

		denominator.Add(&a.Y, &a.Y)
	} else {
		// Compute numerator
		lambda.Sub(&b.Y, &a.Y)

		denominator.Sub(&b.X, &a.X)
	}
	denominator.Inverse(&denominator)
	lambda.Mul(&lambda, &denominator)

	// Compute x_3 as lambda^2 - a_x - b_x
	g.inner.X.Square(&lambda)
	g.inner.X.Sub(&g.inner.X, &a.X)
	g.inner.X.Sub(&g.inner.X, &b.X)

	// Compute y as lambda * (a_x - x_3) - a_y
	g.inner.Y.Sub(&a.X, &g.inner.X)
	g.inner.Y.Mul(&g.inner.Y, &lambda)
	g.inner.Y.Sub(&g.inner.Y, &a.Y)
}
