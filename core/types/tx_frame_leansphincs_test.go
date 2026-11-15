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

package types

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/leansphincs"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestFrameTxLeanSphincs(t *testing.T) {
	seed := bytes.Repeat([]byte{0x42}, leansphincs.SeedSize)
	pub, err := leansphincs.KeyGen(seed)
	if err != nil {
		t.Fatal(err)
	}
	digest := common.Hash{0x12}
	sig, err := leansphincs.Sign(seed, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	entry := SignatureEntry{Scheme: FrameTxSchemeLeanSphincs, Signer: pub, Signature: sig}
	if err := ValidateFrameTxSignatures(SignatureList{entry}, common.Address{}, digest); err != nil {
		t.Fatalf("canonical digest: %v", err)
	}
	entry.Msg = digest[:]
	if err := ValidateFrameTxSignatures(SignatureList{entry}, common.Address{}, common.Hash{0xff}); err != nil {
		t.Fatalf("explicit digest: %v", err)
	}
	wrongDigest := common.Hash{0x13}
	for name, bad := range map[string]SignatureEntry{
		"wrong digest":    {Scheme: entry.Scheme, Signer: pub, Msg: wrongDigest[:], Signature: sig},
		"short signer":    {Scheme: entry.Scheme, Signer: pub[:31], Msg: digest[:], Signature: sig},
		"long signer":     {Scheme: entry.Scheme, Signer: append(bytes.Clone(pub), 0), Msg: digest[:], Signature: sig},
		"short signature": {Scheme: entry.Scheme, Signer: pub, Msg: digest[:], Signature: sig[:len(sig)-1]},
		"zero digest":     {Scheme: entry.Scheme, Signer: pub, Msg: make([]byte, 32), Signature: sig},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateFrameTxSignatures(SignatureList{bad}, common.Address{}, digest); err == nil {
				t.Fatal("accepted invalid signature")
			}
		})
	}
	frame := Frame{Mode: ModeDefault, Value: new(uint256.Int)}
	tx := &FrameTx{
		ChainID: uint256.NewInt(1), Frames: []Frame{frame}, Signatures: SignatureList{entry},
		Fees: Fees{MaxFee: new(uint256.Int), MaxPriorityFeePerGas: new(uint256.Int)},
	}
	if err := tx.ValidateStatic(); err != nil {
		t.Fatalf("valid static frame: %v", err)
	}
	for _, size := range []int{0, 20, 31, 33} {
		tx.Signatures[0].Signer = make([]byte, size)
		if err := tx.ValidateStatic(); err == nil {
			t.Fatalf("accepted %d-byte signer", size)
		}
	}
	// The key must never be truncated into an address.
	if got := entry.ResolvedSigner(common.Address{0xaa}); got != (common.Address{}) {
		t.Fatalf("ResolvedSigner of a leanSPHINCS entry: got %x, want the zero address", got)
	}
	if got := FrameTxSignatureGas(&entry); got != params.FrameTxLeanSphincsSigGas {
		t.Fatalf("verification gas: got %d", got)
	}
	entry.Msg = nil
	entry.Signature = make([]byte, leansphincs.SignatureSize)
	base, err := FrameTxIntrinsicGas(nil, nil, common.Address{}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	gas, err := FrameTxIntrinsicGas(nil, SignatureList{entry}, common.Address{}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	// A canonical signature commits to its length, so its bytes are priced
	// uniformly as non-zero, even when they are all zero.
	want := base + params.FrameTxLeanSphincsSigGas +
		uint64(leansphincs.SignatureSize)*params.TxDataNonZeroGasEIP2028
	for _, b := range pub {
		if b == 0 {
			want += params.TxDataZeroGas
		} else {
			want += params.TxDataNonZeroGasEIP2028
		}
	}
	if gas != want {
		t.Fatalf("canonical signature gas: got %d, want %d", gas, want)
	}
	entry.Msg = digest[:]
	gas, err = FrameTxIntrinsicGas(nil, SignatureList{entry}, common.Address{}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if gas >= want {
		t.Fatalf("explicit zero-filled signature cost: got %d, canonical %d", gas, want)
	}
}
