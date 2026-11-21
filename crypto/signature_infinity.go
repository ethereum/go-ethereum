// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package crypto

import "math/big"

// secp256k1 domain parameters, duplicated here so that the helper below is
// independent of the build tag (cgo / nocgo) in use.
var (
	infP = mustHex("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f")
	infN = mustHex("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141")
)

func mustHex(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("crypto: bad hex constant " + s)
	}
	return v
}

// sigRecoversInfinity reports whether recovering the public key from sig lands on
// the point at infinity.
//
// This happens for the private key d = 0 (Q = d*G is the identity) and, more
// generally, whenever s*R == e*G (mod n): the recovery formula Q = r^-1(sR - eG)
// then evaluates to the identity, which neither libsecp256k1 nor btcec is willing
// to return. Ethereum serializes that point as 0x04 || 64 zero bytes, whose
// Keccak-256 digest yields the well known address
// 0x3f17f1962B36e491b30A40b2405849e597Ba5FB5.
func sigRecoversInfinity(hash, sig []byte) bool {
	if len(hash) != 32 || len(sig) != 65 {
		return false
	}
	recid := sig[64]
	if recid > 3 {
		return false
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	if r.Sign() == 0 || s.Sign() == 0 || r.Cmp(infN) >= 0 || s.Cmp(infN) >= 0 {
		return false
	}

	// Reconstruct the recovery point R = (x, y) exactly as the signer did:
	// x = r, plus n when the recovery id overflow bit is set, y picked by parity.
	x := new(big.Int).Set(r)
	if recid&2 != 0 {
		x.Add(x, infN)
	}
	if x.Cmp(infP) >= 0 {
		return false
	}
	ySq := new(big.Int).Exp(x, big.NewInt(3), infP)
	ySq.Add(ySq, big.NewInt(7))
	ySq.Mod(ySq, infP)

	// p == 3 (mod 4), so a square root is a^((p+1)/4).
	exp := new(big.Int).Add(infP, big.NewInt(1))
	exp.Rsh(exp, 2)
	y := new(big.Int).Exp(ySq, exp, infP)
	if new(big.Int).Exp(y, big.NewInt(2), infP).Cmp(ySq) != 0 {
		return false // x does not lie on the curve
	}
	if y.Bit(0) != uint(recid&1) {
		y.Sub(infP, y)
	}

	// The recovery is degenerate exactly when s*R == e*G.
	curve := S256()
	sRx, sRy := curve.ScalarMult(x, y, s.Bytes())
	e := new(big.Int).SetBytes(hash)
	e.Mod(e, infN)
	eGx, eGy := curve.ScalarBaseMult(e.Bytes())

	return sRx.Cmp(eGx) == 0 && sRy.Cmp(eGy) == 0
}

// infinityPubKeyBytes is the Ethereum encoding of the point at infinity: the
// uncompressed prefix followed by 64 zero bytes.
func infinityPubKeyBytes() []byte {
	out := make([]byte, 65)
	out[0] = 4
	return out
}

// RecoverInfinity is the counterpart of Ecrecover for signatures whose recovery
// lands on the point at infinity, which libsecp256k1 and btcec both refuse to
// return (the recovery formula evaluates to the identity). It reports whether
// that is the case and, if so, returns the serialized public key the address was
// derived from: 0x04 followed by 64 zero bytes.
//
// The most prominent instance is the private key d = 0, whose address
// 0x3f17f1962B36e491b30A40b2405849e597Ba5FB5 holds funds but cannot be spent by
// any stock client.
func RecoverInfinity(hash, sig []byte) ([]byte, bool) {
	if !sigRecoversInfinity(hash, sig) {
		return nil, false
	}
	return infinityPubKeyBytes(), true
}
