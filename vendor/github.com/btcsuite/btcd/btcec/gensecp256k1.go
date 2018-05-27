// Copyright (c) 2014-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// This file is ignored during the regular build due to the following build tag.
// This build tag is set during go generate.
// +build gensecp256k1

package btcec

// References:
//   [GECC]: Guide to Elliptic Curve Cryptography (Hankerson, Menezes, Vanstone)

import (
	"encoding/binary"
	"math/big"
)

// secp256k1BytePoints are dummy points used so the code which generates the
// real values can compile.
var secp256k1BytePoints = ""

// getDoublingPoints returns all the possible G^(2^i) for i in
// 0..n-1 where n is the curve's bit size (256 in the case of secp256k1)
// the coordinates are recorded as Jacobian coordinates.
func (curve *KoblitzCurve) getDoublingPoints() [][3]fieldVal {
	doublingPoints := make([][3]fieldVal, curve.BitSize)

	// initialize px, py, pz to the Jacobian coordinates for the base point
	px, py := curve.bigAffineToField(curve.Gx, curve.Gy)
	pz := new(fieldVal).SetInt(1)
	for i := 0; i < curve.BitSize; i++ {
		doublingPoints[i] = [3]fieldVal{*px, *py, *pz}
		// P = 2*P
		curve.doubleJacobian(px, py, pz, px, py, pz)
	}
	return doublingPoints
}

// SerializedBytePoints returns a serialized byte slice which contains all of
// the possible points per 8-bit window.  This is used to when generating
// secp256k1.go.
func (curve *KoblitzCurve) SerializedBytePoints() []byte {
	doublingPoints := curve.getDoublingPoints()

	// Segregate the bits into byte-sized windows
	serialized := make([]byte, curve.byteSize*256*3*10*4)
	offset := 0
	for byteNum := 0; byteNum < curve.byteSize; byteNum++ {
		// Grab the 8 bits that make up this byte from doublingPoints.
		startingBit := 8 * (curve.byteSize - byteNum - 1)
		computingPoints := doublingPoints[startingBit : startingBit+8]

		// Compute all points in this window and serialize them.
		for i := 0; i < 256; i++ {
			px, py, pz := new(fieldVal), new(fieldVal), new(fieldVal)
			for j := 0; j < 8; j++ {
				if i>>uint(j)&1 == 1 {
					curve.addJacobian(px, py, pz, &computingPoints[j][0],
						&computingPoints[j][1], &computingPoints[j][2], px, py, pz)
				}
			}
			for i := 0; i < 10; i++ {
				binary.LittleEndian.PutUint32(serialized[offset:], px.n[i])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				binary.LittleEndian.PutUint32(serialized[offset:], py.n[i])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				binary.LittleEndian.PutUint32(serialized[offset:], pz.n[i])
				offset += 4
			}
		}
	}

	return serialized
}

// sqrt returns the square root of the provided big integer using Newton's
// method.  It's only compiled and used during generation of pre-computed
// values, so speed is not a huge concern.
func sqrt(n *big.Int) *big.Int {
	// Initial guess = 2^(log_2(n)/2)
	guess := big.NewInt(2)
	guess.Exp(guess, big.NewInt(int64(n.BitLen()/2)), nil)

	// Now refine using Newton's method.
	big2 := big.NewInt(2)
	prevGuess := big.NewInt(0)
	for {
		prevGuess.Set(guess)
		guess.Add(guess, new(big.Int).Div(n, guess))
		guess.Div(guess, big2)
		if guess.Cmp(prevGuess) == 0 {
			break
		}
	}
	return guess
}

// EndomorphismVectors runs the first 3 steps of algorithm 3.74 from [GECC] to
// generate the linearly independent vectors needed to generate a balanced
// length-two representation of a multiplier such that k = k1 + k2λ (mod N) and
// returns them.  Since the values will always be the same given the fact that N
// and λ are fixed, the final results can be accelerated by storing the
// precomputed values with the curve.
func (curve *KoblitzCurve) EndomorphismVectors() (a1, b1, a2, b2 *big.Int) {
	bigMinus1 := big.NewInt(-1)

	// This section uses an extended Euclidean algorithm to generate a
	// sequence of equations:
	//  s[i] * N + t[i] * λ = r[i]

	nSqrt := sqrt(curve.N)
	u, v := new(big.Int).Set(curve.N), new(big.Int).Set(curve.lambda)
	x1, y1 := big.NewInt(1), big.NewInt(0)
	x2, y2 := big.NewInt(0), big.NewInt(1)
	q, r := new(big.Int), new(big.Int)
	qu, qx1, qy1 := new(big.Int), new(big.Int), new(big.Int)
	s, t := new(big.Int), new(big.Int)
	ri, ti := new(big.Int), new(big.Int)
	a1, b1, a2, b2 = new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	found, oneMore := false, false
	for u.Sign() != 0 {
		// q = v/u
		q.Div(v, u)

		// r = v - q*u
		qu.Mul(q, u)
		r.Sub(v, qu)

		// s = x2 - q*x1
		qx1.Mul(q, x1)
		s.Sub(x2, qx1)

		// t = y2 - q*y1
		qy1.Mul(q, y1)
		t.Sub(y2, qy1)

		// v = u, u = r, x2 = x1, x1 = s, y2 = y1, y1 = t
		v.Set(u)
		u.Set(r)
		x2.Set(x1)
		x1.Set(s)
		y2.Set(y1)
		y1.Set(t)

		// As soon as the remainder is less than the sqrt of n, the
		// values of a1 and b1 are known.
		if !found && r.Cmp(nSqrt) < 0 {
			// When this condition executes ri and ti represent the
			// r[i] and t[i] values such that i is the greatest
			// index for which r >= sqrt(n).  Meanwhile, the current
			// r and t values are r[i+1] and t[i+1], respectively.

			// a1 = r[i+1], b1 = -t[i+1]
			a1.Set(r)
			b1.Mul(t, bigMinus1)
			found = true
			oneMore = true

			// Skip to the next iteration so ri and ti are not
			// modified.
			continue

		} else if oneMore {
			// When this condition executes ri and ti still
			// represent the r[i] and t[i] values while the current
			// r and t are r[i+2] and t[i+2], respectively.

			// sum1 = r[i]^2 + t[i]^2
			rSquared := new(big.Int).Mul(ri, ri)
			tSquared := new(big.Int).Mul(ti, ti)
			sum1 := new(big.Int).Add(rSquared, tSquared)

			// sum2 = r[i+2]^2 + t[i+2]^2
			r2Squared := new(big.Int).Mul(r, r)
			t2Squared := new(big.Int).Mul(t, t)
			sum2 := new(big.Int).Add(r2Squared, t2Squared)

			// if (r[i]^2 + t[i]^2) <= (r[i+2]^2 + t[i+2]^2)
			if sum1.Cmp(sum2) <= 0 {
				// a2 = r[i], b2 = -t[i]
				a2.Set(ri)
				b2.Mul(ti, bigMinus1)
			} else {
				// a2 = r[i+2], b2 = -t[i+2]
				a2.Set(r)
				b2.Mul(t, bigMinus1)
			}

			// All done.
			break
		}

		ri.Set(r)
		ti.Set(t)
	}

	return a1, b1, a2, b2
}
