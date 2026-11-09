// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

//go:build zisk

package crypto

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/0xPolygonHermez/zisk/ziskasm/lang/go"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// Signature recovery and verification by the ZisK zkVM's secp256k1 routines
// (the EF zkVM accelerator ABI, package zkvm of ziskasm/lang/go). Besides
// being far cheaper to prove, this keeps the pure-Go path's precomputed
// generator table out of the picture: decompressing it costs more than a
// billion zkVM steps on first use.

// Ecrecover returns the uncompressed public key that created the given signature.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	if len(sig) != SignatureLength {
		return nil, errors.New("invalid signature")
	}
	if len(hash) != DigestLength {
		return nil, fmt.Errorf("hash is required to be exactly %d bytes (%d)", DigestLength, len(hash))
	}
	pub, err := zkvm.Secp256k1Ecrecover((*[32]byte)(hash), (*[64]byte)(sig[:64]), sig[RecoveryIDOffset])
	if err != nil {
		return nil, errors.New("invalid signature")
	}
	out := make([]byte, 65)
	out[0] = 4 // uncompressed point
	copy(out[1:], pub[:])
	return out, nil
}

// SigToPub returns the public key that created the given signature.
func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	pub, err := Ecrecover(hash, sig)
	if err != nil {
		return nil, err
	}
	return &ecdsa.PublicKey{
		Curve: S256(),
		X:     new(big.Int).SetBytes(pub[1:33]),
		Y:     new(big.Int).SetBytes(pub[33:]),
	}, nil
}

// VerifySignature checks that the given public key created signature over hash.
// The public key should be in compressed (33 bytes) or uncompressed (65 bytes) format.
// The signature should have the 64 byte [R || S] format.
func VerifySignature(pubkey, hash, signature []byte) bool {
	if len(signature) != 64 || len(hash) != DigestLength {
		return false
	}
	// Reject malleable signatures, as libsecp256k1 does.
	var s secp256k1.ModNScalar
	if s.SetByteSlice(signature[32:]) || s.IsOverHalfOrder() {
		return false
	}
	key, err := secp256k1.ParsePubKey(pubkey)
	if err != nil {
		return false
	}
	var xy [64]byte
	key.X().FillBytes(xy[:32])
	key.Y().FillBytes(xy[32:])
	ok, err := zkvm.Secp256k1Verify((*[32]byte)(hash), (*[64]byte)(signature), &xy)
	return err == nil && ok
}
