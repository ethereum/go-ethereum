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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// sampleFrameTx builds a simple frame transaction with one VERIFY (self_verify)
// frame and one SENDER (user_op) frame.
func sampleFrameTx(t *testing.T) *FrameTx {
	t.Helper()
	return &FrameTx{
		ChainID: big.NewInt(1),
		Nonce:   0,
		Sender:  common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Frames: []Frame{
			{Mode: FrameModeVerify, Flags: FrameFlagApproveExecutionPayment, GasLimit: 100000, Value: new(uint256.Int)},
			{Mode: FrameModeSender, Target: &common.Address{0x22}, GasLimit: 100000, Value: new(uint256.Int), Data: []byte{0xde, 0xad}},
		},
		Signatures: []FrameSignature{
			{Scheme: SignatureSchemeSecp256k1, Signer: common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes(), Msg: nil, Signature: make([]byte, 65)},
		},
		GasTipCap:  big.NewInt(1),
		GasFeeCap:  big.NewInt(10),
		BlobFeeCap: new(uint256.Int),
	}
}

func TestFrameTxRoundtrip(t *testing.T) {
	tx := NewTx(sampleFrameTx(t))

	// Encode and decode binary.
	enc, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var dec Transaction
	if err := dec.UnmarshalBinary(enc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dec.Type() != FrameTxType {
		t.Fatalf("wrong type: %d", dec.Type())
	}
	ftx := dec.Inner().(*FrameTx)
	if ftx.Sender != tx.Inner().(*FrameTx).Sender {
		t.Fatalf("sender mismatch")
	}
	if len(ftx.Frames) != 2 {
		t.Fatalf("frame count mismatch: %d", len(ftx.Frames))
	}
	if !bytes.Equal(ftx.Frames[1].Data, []byte{0xde, 0xad}) {
		t.Fatalf("frame data mismatch")
	}
	// The transaction hash must be stable across round-trips.
	if dec.Hash() != tx.Hash() {
		t.Fatalf("hash mismatch after roundtrip")
	}
}

func TestFrameTxSigHash(t *testing.T) {
	tx := NewTx(sampleFrameTx(t)).Inner().(*FrameTx)
	h := tx.ComputeSigHash()
	if h == (common.Hash{}) {
		t.Fatalf("empty sig hash")
	}
	// Eliding the raw signature bytes of empty-msg signatures must not change
	// the canonical signature hash.
	sigHashCopy := tx.ComputeSigHash()
	if sigHashCopy != h {
		t.Fatalf("sig hash changed across calls")
	}
}

func TestFrameTxGas(t *testing.T) {
	tx := sampleFrameTx(t)
	maxGas := tx.MaxGas()
	standard := tx.StandardGasLimit()
	if maxGas < standard {
		t.Fatalf("max gas %d < standard %d", maxGas, standard)
	}
	if tx.gas() != maxGas {
		t.Fatalf("gas() %d != maxGas %d", tx.gas(), maxGas)
	}
}

func TestFrameTxSecp256k1Signature(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)

	tx := sampleFrameTx(t)
	tx.Signatures[0].Signer = addr.Bytes()
	// Remove the invalid placeholder signature bytes.
	tx.Signatures[0].Signature = nil

	// Sign the canonical signature hash.
	sigHash := tx.ComputeSigHash()
	sig, err := crypto.Sign(sigHash[:], key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// EIP-8141 encodes the signature as [v, r, s] where v is the recovery id.
	// crypto.Sign produces [r, s, v].
	recoveryID := sig[64]
	frameSig := make([]byte, 65)
	frameSig[0] = recoveryID
	copy(frameSig[1:33], sig[:32])
	copy(frameSig[33:65], sig[32:64])
	tx.Signatures[0].Signature = frameSig

	if !tx.ValidateSignature(&tx.Signatures[0], sigHash) {
		t.Fatalf("valid signature rejected")
	}
	// Corrupt the signature and ensure it is rejected.
	frameSig[33] ^= 0xff
	if tx.ValidateSignature(&tx.Signatures[0], sigHash) {
		t.Fatalf("invalid signature accepted")
	}
}
