// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package crypto

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// zeroKeyAddr is the address of the private key d = 0. Its public key Q = d*G is
// the point at infinity, serialized as 0x04 followed by 64 zero bytes; the
// Keccak-256 digest of that serialization is this address.
const zeroKeyAddrHex = "0x3f17f1962B36e491b30A40b2405849e597Ba5FB5"

// signWithZeroKey produces an ECDSA signature with the private key d = 0, i.e.
// with the r/s equations reduced to s = e*k^-1 (mod n). The key cannot be passed
// to Sign, because its public key is the point at infinity and ecdsa.PublicKey
// cannot express that.
func signWithZeroKey(t *testing.T, hash []byte) []byte {
	t.Helper()
	n := S256().Params().N
	for k := uint64(1); ; k++ {
		kb := new(big.Int).SetUint64(k)
		x, y := S256().ScalarBaseMult(kb.Bytes())
		r := new(big.Int).Mod(x, n)
		if r.Sign() == 0 {
			continue
		}
		e := new(big.Int).SetBytes(hash)
		e.Mod(e, n)
		s := new(big.Int).Mul(e, new(big.Int).ModInverse(kb, n))
		s.Mod(s, n)
		if s.Sign() == 0 {
			continue
		}
		recid := byte(y.Bit(0))
		if s.Cmp(new(big.Int).Rsh(n, 1)) > 0 {
			s.Sub(n, s)
			recid ^= 1
		}
		sig := make([]byte, SignatureLength)
		r.FillBytes(sig[:32])
		s.FillBytes(sig[32:64])
		sig[64] = recid
		return sig
	}
}

// TestEcrecoverInfinity checks that a signature made with the private key d = 0
// recovers to the serialization of the point at infinity. Neither libsecp256k1
// nor btcec returns that point: the first errors out, the second silently yields
// an unrelated key, so the same signature used to resolve differently depending
// on the build tag.
func TestEcrecoverInfinity(t *testing.T) {
	hash := Keccak256([]byte("the point at infinity"))
	sig := signWithZeroKey(t, hash)

	pub, err := Ecrecover(hash, sig)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	want := make([]byte, 65)
	want[0] = 4
	if string(pub) != string(want) {
		t.Fatalf("wrong public key: have %x, want %x", pub, want)
	}
	// Derive the address the way the signers do - straight from the bytes, as
	// UnmarshalPubkey rejects the all-zero point (it is not on the curve).
	got := common.BytesToAddress(Keccak256(pub[1:])[12:])
	if wantAddr := common.HexToAddress(zeroKeyAddrHex); got != wantAddr {
		t.Fatalf("wrong address: have %s, want %s", got, wantAddr)
	}
}

// TestUnmarshalPubkeyInfinity documents that the zero point is not on the curve,
// so the generic public key parser keeps rejecting it.
func TestUnmarshalPubkeyInfinity(t *testing.T) {
	pub := make([]byte, 65)
	pub[0] = 4
	if _, err := UnmarshalPubkey(pub); err == nil {
		t.Fatal("expected UnmarshalPubkey to reject the point at infinity")
	}
}

// TestEcrecoverRegular guards that ordinary signatures keep recovering as before.
func TestEcrecoverRegular(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	hash := Keccak256([]byte("ordinary signature"))
	sig, err := Sign(hash, key)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := Ecrecover(hash, sig)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := common.BytesToAddress(Keccak256(pub[1:])[12:]), PubkeyToAddress(key.PublicKey); got != want {
		t.Fatalf("wrong signer: have %s, want %s", got, want)
	}
}
