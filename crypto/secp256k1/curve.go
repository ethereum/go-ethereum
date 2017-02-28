// Copyright 2010 The Go Authors. All rights reserved.
// Copyright 2011 ThePiachu. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright
//   notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
//   copyright notice, this list of conditions and the following disclaimer
//   in the documentation and/or other materials provided with the
//   distribution.
// * Neither the name of Google Inc. nor the names of its
//   contributors may be used to endorse or promote products derived from
//   this software without specific prior written permission.
// * The name of ThePiachu may not be used to endorse or promote products
//   derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package secp256k1

import (
	"crypto/elliptic"
	"math/big"
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/common/math"
)

/*
#include "libsecp256k1/include/secp256k1.h"
extern int secp256k1_pubkey_scalar_mul(const secp256k1_context* ctx, const unsigned char *point, const unsigned char *scalar);
*/
import "C"

// This code is from https://github.com/ThePiachu/GoBit and implements
// several Koblitz elliptic curves over prime fields.
//
// The curve methods, internally, on Jacobian coordinates. For a given
// (x, y) position on the curve, the Jacobian coordinates are (x1, y1,
// z1) where x = x1/z1² and y = y1/z1³. The greatest speedups come
// when the whole calculation can be performed within the transform
// (as in ScalarMult and ScalarBaseMult). But even for Add and Double,
// it's faster to apply and reverse the transform than to operate in
// affine coordinates.

// A BitCurve represents a Koblitz Curve with a=0.
// See http://www.hyperelliptic.org/EFD/g1p/auto-shortw.html
type BitCurve struct {
	P       *big.Int // the order of the underlying field
	N       *big.Int // the order of the base point
	B       *big.Int // the constant of the BitCurve equation
	Gx, Gy  *big.Int // (x,y) of the base point
	BitSize int      // the size of the underlying field
}

func (BitCurve *BitCurve) Params() *elliptic.CurveParams {
	return &elliptic.CurveParams{
		P:       BitCurve.P,
		N:       BitCurve.N,
		B:       BitCurve.B,
		Gx:      BitCurve.Gx,
		Gy:      BitCurve.Gy,
		BitSize: BitCurve.BitSize,
	}
}

// IsOnBitCurve returns true if the given (x,y) lies on the BitCurve.
func (BitCurve *BitCurve) IsOnCurve(x, y *big.Int) bool {
	// y² = x³ + b
	y2 := new(big.Int).Mul(y, y) //y²
	y2.Mod(y2, BitCurve.P)       //y²%P

	x3 := new(big.Int).Mul(x, x) //x²
	x3.Mul(x3, x)                //x³

	x3.Add(x3, BitCurve.B) //x³+B
	x3.Mod(x3, BitCurve.P) //(x³+B)%P

	return x3.Cmp(y2) == 0
}

//TODO: double check if the function is okay
// affineFromJacobian reverses the Jacobian transform. See the comment at the
// top of the file.
func (BitCurve *BitCurve) affineFromJacobian(x, y, z *big.Int) (xOut, yOut *big.Int) {
	zinv := new(big.Int).ModInverse(z, BitCurve.P)
	zinvsq := new(big.Int).Mul(zinv, zinv)

	xOut = new(big.Int).Mul(x, zinvsq)
	xOut.Mod(xOut, BitCurve.P)
	zinvsq.Mul(zinvsq, zinv)
	yOut = new(big.Int).Mul(y, zinvsq)
	yOut.Mod(yOut, BitCurve.P)
	return
}

// Add returns the sum of (x1,y1) and (x2,y2)
func (BitCurve *BitCurve) Add(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {
	z := new(big.Int).SetInt64(1)
	return BitCurve.affineFromJacobian(BitCurve.addJacobian(x1, y1, z, x2, y2, z))
}

// addJacobian takes two points in Jacobian coordinates, (x1, y1, z1) and
// (x2, y2, z2) and returns their sum, also in Jacobian form.
func (BitCurve *BitCurve) addJacobian(x1, y1, z1, x2, y2, z2 *big.Int) (*big.Int, *big.Int, *big.Int) {
	// See http://hyperelliptic.org/EFD/g1p/auto-shortw-jacobian-0.html#addition-add-2007-bl
	z1z1 := new(big.Int).Mul(z1, z1)
	z1z1.Mod(z1z1, BitCurve.P)
	z2z2 := new(big.Int).Mul(z2, z2)
	z2z2.Mod(z2z2, BitCurve.P)

	u1 := new(big.Int).Mul(x1, z2z2)
	u1.Mod(u1, BitCurve.P)
	u2 := new(big.Int).Mul(x2, z1z1)
	u2.Mod(u2, BitCurve.P)
	h := new(big.Int).Sub(u2, u1)
	if h.Sign() == -1 {
		h.Add(h, BitCurve.P)
	}
	i := new(big.Int).Lsh(h, 1)
	i.Mul(i, i)
	j := new(big.Int).Mul(h, i)

	s1 := new(big.Int).Mul(y1, z2)
	s1.Mul(s1, z2z2)
	s1.Mod(s1, BitCurve.P)
	s2 := new(big.Int).Mul(y2, z1)
	s2.Mul(s2, z1z1)
	s2.Mod(s2, BitCurve.P)
	r := new(big.Int).Sub(s2, s1)
	if r.Sign() == -1 {
		r.Add(r, BitCurve.P)
	}
	r.Lsh(r, 1)
	v := new(big.Int).Mul(u1, i)

	x3 := new(big.Int).Set(r)
	x3.Mul(x3, x3)
	x3.Sub(x3, j)
	x3.Sub(x3, v)
	x3.Sub(x3, v)
	x3.Mod(x3, BitCurve.P)

	y3 := new(big.Int).Set(r)
	v.Sub(v, x3)
	y3.Mul(y3, v)
	s1.Mul(s1, j)
	s1.Lsh(s1, 1)
	y3.Sub(y3, s1)
	y3.Mod(y3, BitCurve.P)

	z3 := new(big.Int).Add(z1, z2)
	z3.Mul(z3, z3)
	z3.Sub(z3, z1z1)
	if z3.Sign() == -1 {
		z3.Add(z3, BitCurve.P)
	}
	z3.Sub(z3, z2z2)
	if z3.Sign() == -1 {
		z3.Add(z3, BitCurve.P)
	}
	z3.Mul(z3, h)
	z3.Mod(z3, BitCurve.P)

	return x3, y3, z3
}

// Double returns 2*(x,y)
func (BitCurve *BitCurve) Double(x1, y1 *big.Int) (*big.Int, *big.Int) {
	z1 := new(big.Int).SetInt64(1)
	return BitCurve.affineFromJacobian(BitCurve.doubleJacobian(x1, y1, z1))
}

// doubleJacobian takes a point in Jacobian coordinates, (x, y, z), and
// returns its double, also in Jacobian form.
func (BitCurve *BitCurve) doubleJacobian(x, y, z *big.Int) (*big.Int, *big.Int, *big.Int) {
	// See http://hyperelliptic.org/EFD/g1p/auto-shortw-jacobian-0.html#doubling-dbl-2009-l

	a := new(big.Int).Mul(x, x) //X1²
	b := new(big.Int).Mul(y, y) //Y1²
	c := new(big.Int).Mul(b, b) //B²

	d := new(big.Int).Add(x, b) //X1+B
	d.Mul(d, d)                 //(X1+B)²
	d.Sub(d, a)                 //(X1+B)²-A
	d.Sub(d, c)                 //(X1+B)²-A-C
	d.Mul(d, big.NewInt(2))     //2*((X1+B)²-A-C)

	e := new(big.Int).Mul(big.NewInt(3), a) //3*A
	f := new(big.Int).Mul(e, e)             //E²

	x3 := new(big.Int).Mul(big.NewInt(2), d) //2*D
	x3.Sub(f, x3)                            //F-2*D
	x3.Mod(x3, BitCurve.P)

	y3 := new(big.Int).Sub(d, x3)                  //D-X3
	y3.Mul(e, y3)                                  //E*(D-X3)
	y3.Sub(y3, new(big.Int).Mul(big.NewInt(8), c)) //E*(D-X3)-8*C
	y3.Mod(y3, BitCurve.P)

	z3 := new(big.Int).Mul(y, z) //Y1*Z1
	z3.Mul(big.NewInt(2), z3)    //3*Y1*Z1
	z3.Mod(z3, BitCurve.P)

	return x3, y3, z3
}

func (BitCurve *BitCurve) ScalarMult(Bx, By *big.Int, scalar []byte) (*big.Int, *big.Int) {
	// Ensure scalar is exactly 32 bytes. We pad always, even if
	// scalar is 32 bytes long, to avoid a timing side channel.
	if len(scalar) > 32 {
		panic("can't handle scalars > 256 bits")
	}
	// NOTE: potential timing issue
	padded := make([]byte, 32)
	copy(padded[32-len(scalar):], scalar)
	scalar = padded

	// Do the multiplication in C, updating point.
	point := make([]byte, 64)
	math.ReadBits(Bx, point[:32])
	math.ReadBits(By, point[32:])
	pointPtr := (*C.uchar)(unsafe.Pointer(&point[0]))
	scalarPtr := (*C.uchar)(unsafe.Pointer(&scalar[0]))
	res := C.secp256k1_pubkey_scalar_mul(context, pointPtr, scalarPtr)

	// Unpack the result and clear temporaries.
	x := new(big.Int).SetBytes(point[:32])
	y := new(big.Int).SetBytes(point[32:])
	for i := range point {
		point[i] = 0
	}
	for i := range padded {
		scalar[i] = 0
	}
	if res != 1 {
		return nil, nil
	}
	return x, y
}

// ScalarBaseMult returns k*G, where G is the base point of the group and k is
// an integer in big-endian form.
func (BitCurve *BitCurve) ScalarBaseMult(k []byte) (*big.Int, *big.Int) {
	return BitCurve.ScalarMult(BitCurve.Gx, BitCurve.Gy, k)
}

// Marshal converts a point into the form specified in section 4.3.6 of ANSI
// X9.62.
func (BitCurve *BitCurve) Marshal(x, y *big.Int) []byte {
	byteLen := (BitCurve.BitSize + 7) >> 3

	ret := make([]byte, 1+2*byteLen)
	ret[0] = 4 // uncompressed point

	xBytes := x.Bytes()
	copy(ret[1+byteLen-len(xBytes):], xBytes)
	yBytes := y.Bytes()
	copy(ret[1+2*byteLen-len(yBytes):], yBytes)
	return ret
}

// Unmarshal converts a point, serialised by Marshal, into an x, y pair. On
// error, x = nil.
func (BitCurve *BitCurve) Unmarshal(data []byte) (x, y *big.Int) {
	byteLen := (BitCurve.BitSize + 7) >> 3
	if len(data) != 1+2*byteLen {
		return
	}
	if data[0] != 4 { // uncompressed form
		return
	}
	x = new(big.Int).SetBytes(data[1 : 1+byteLen])
	y = new(big.Int).SetBytes(data[1+byteLen:])
	return
}

var (
	initonce sync.Once
	theCurve *BitCurve
)

// S256 returns a BitCurve which implements secp256k1 (see SEC 2 section 2.7.1)
func S256() *BitCurve {
	initonce.Do(func() {
		// See SEC 2 section 2.7.1
		// curve parameters taken from:
		// http://www.secg.org/collateral/sec2_final.pdf
		theCurve = new(BitCurve)
		theCurve.P, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F", 16)
		theCurve.N, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
		theCurve.B, _ = new(big.Int).SetString("0000000000000000000000000000000000000000000000000000000000000007", 16)
		theCurve.Gx, _ = new(big.Int).SetString("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798", 16)
		theCurve.Gy, _ = new(big.Int).SetString("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8", 16)
		theCurve.BitSize = 256
	})
	return theCurve
}
