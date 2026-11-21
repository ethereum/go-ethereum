// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package types

import (
	"testing"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// TestSetCodeAuthorizationInfinity covers an EIP-7702 authorisation whose
// signature recovers to the point at infinity, i.e. one made with the private
// key d = 0. Stock clients fail to recover it and silently skip the tuple, which
// leaves the delegating address unusable even though the wrapping transaction is
// perfectly valid.
func TestSetCodeAuthorizationInfinity(t *testing.T) {
	var (
		zeroAddr = common.HexToAddress("0x3f17f1962B36e491b30A40b2405849e597Ba5FB5")
		delegate = common.HexToAddress("0x75692E34c555A10a3D9FaF99B745d67a42c0Bca5")
		auth     = SetCodeAuthorization{ChainID: *uint256.NewInt(1), Address: delegate}
	)
	auth.V = 1
	auth.R.SetFromHex("0xe9172dd3a40bd5c987b16007901e0c2ed3120ad2f33b7023073c112373103430")
	auth.S.SetFromHex("0x37e465eeaa3c911d0d243f4e86d53db0da6af4ea9eb4df895f3fa0c7cb35f238")

	wantHash := common.HexToHash("0x540378ada450cd21f371178cd341bd536400083d6b8eaad364794877e4a6ad2e")
	if hash := auth.SigHash(); hash != wantHash {
		t.Fatalf("wrong sighash: have %x, want %x", hash, wantHash)
	}
	if !crypto.ValidateSignatureValues(auth.V, auth.R.ToBig(), auth.S.ToBig(), true) {
		t.Fatal("signature values rejected before recovery")
	}
	got, err := auth.Authority()
	if err != nil {
		t.Fatalf("authority recovery failed: %v", err)
	}
	if got != zeroAddr {
		t.Fatalf("wrong authority: have %s, want %s", got, zeroAddr)
	}
}

// TestSetCodeAuthorizationRegular guards that the fallback does not disturb
// ordinary signatures.
func TestSetCodeAuthorizationRegular(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	delegate := common.HexToAddress("0x75692E34c555A10a3D9FaF99B745d67a42c0Bca5")
	auth := SetCodeAuthorization{ChainID: *uint256.NewInt(1), Address: delegate, Nonce: 7}
	hash := auth.SigHash()
	sig, err := crypto.Sign(hash[:], key)
	if err != nil {
		t.Fatal(err)
	}
	auth.R.SetBytes(sig[:32])
	auth.S.SetBytes(sig[32:64])
	auth.V = sig[64]
	got, err := auth.Authority()
	if err != nil {
		t.Fatal(err)
	}
	if want := crypto.PubkeyToAddress(key.PublicKey); got != want {
		t.Fatalf("wrong authority: have %s, want %s", got, want)
	}
}
