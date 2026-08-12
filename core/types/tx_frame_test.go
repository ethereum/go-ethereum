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
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
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

// TestFrameTxSigHashElision checks that the raw bytes of an empty-msg signature
// are elided from the canonical signature hash (they cannot commit to
// themselves), while the bytes of an explicit-msg signature are committed to.
func TestFrameTxSigHashElision(t *testing.T) {
	tx := sampleFrameTx(t)
	h := tx.ComputeSigHash()
	if h == (common.Hash{}) {
		t.Fatalf("empty sig hash")
	}
	// The sample's only signature has an empty msg, so rewriting its raw bytes
	// must leave the hash untouched.
	tx.Signatures[0].Signature = bytes.Repeat([]byte{0xab}, 65)
	if got := tx.ComputeSigHash(); got != h {
		t.Fatalf("sig hash changed after rewriting elided signature bytes: %x != %x", got, h)
	}
	// Eliding must not mutate the transaction itself.
	if len(tx.Signatures[0].Signature) != 65 {
		t.Fatalf("ComputeSigHash mutated the transaction")
	}
	// With an explicit 32-byte msg the bytes are committed to, so the same
	// rewrite must change the hash.
	tx.Signatures[0].Msg = bytes.Repeat([]byte{0x01}, 32)
	withMsg := tx.ComputeSigHash()
	tx.Signatures[0].Signature = bytes.Repeat([]byte{0xcd}, 65)
	if got := tx.ComputeSigHash(); got == withMsg {
		t.Fatalf("sig hash unchanged after rewriting committed signature bytes")
	}
}

// TestFrameTxGasLimits pins the EIP-8141 gas formulas, in particular that the
// standard calldata cost is STANDARD_TOKEN_COST (4) per token and not the
// per-non-zero-byte price.
func TestFrameTxGasLimits(t *testing.T) {
	tx := &FrameTx{
		ChainID: big.NewInt(1),
		Frames: []Frame{
			// 1 zero byte + 2 non-zero bytes = 1 + 2*4 = 9 tokens.
			{Mode: FrameModeSender, GasLimit: 1000, Value: new(uint256.Int), Data: []byte{0x00, 0x01, 0x02}},
		},
		GasTipCap:  big.NewInt(1),
		GasFeeCap:  big.NewInt(10),
		BlobFeeCap: new(uint256.Int),
	}
	const (
		tokens = 1 + 2*4
		base   = params.FrameTxIntrinsicCost + params.FrameTxPerFrameCost // no signatures
	)
	standard, floor, maxGas, err := tx.GasLimits()
	if err != nil {
		t.Fatalf("GasLimits: %v", err)
	}
	if want := uint64(base + tokens*params.TxDataZeroGas + 1000); standard != want {
		t.Errorf("standard gas = %d, want %d", standard, want)
	}
	if want := uint64(base + tokens*params.TxCostFloorPerToken); floor != want {
		t.Errorf("floor gas = %d, want %d", floor, want)
	}
	if want := max(standard, floor); maxGas != want {
		t.Errorf("max gas = %d, want %d", maxGas, want)
	}
	if tx.gas() != maxGas {
		t.Errorf("gas() = %d, want %d", tx.gas(), maxGas)
	}
}

// TestFrameTxGasOverflow checks that frame gas limits summing past 2^64 are
// reported rather than silently wrapping to a small max_gas.
func TestFrameTxGasOverflow(t *testing.T) {
	tx := &FrameTx{
		ChainID: big.NewInt(1),
		Frames: []Frame{
			{Mode: FrameModeSender, GasLimit: math.MaxUint64, Value: new(uint256.Int)},
			{Mode: FrameModeSender, GasLimit: 1, Value: new(uint256.Int)},
		},
		GasTipCap:  big.NewInt(1),
		GasFeeCap:  big.NewInt(10),
		BlobFeeCap: new(uint256.Int),
	}
	if _, _, _, err := tx.GasLimits(); !errors.Is(err, ErrFrameGasOverflow) {
		t.Fatalf("GasLimits err = %v, want %v", err, ErrFrameGasOverflow)
	}
	if _, err := tx.SumFrameGas(); !errors.Is(err, ErrFrameGasOverflow) {
		t.Fatalf("SumFrameGas err = %v, want %v", err, ErrFrameGasOverflow)
	}
	// gas() cannot report an error, so it must saturate rather than wrap: a
	// wrapped value would sail through the block gas limit check.
	if got := tx.gas(); got != math.MaxUint64 {
		t.Fatalf("gas() = %d, want saturation to MaxUint64", got)
	}
}

// TestFrameTxJSONSignerRoundtrip checks that an absent signer (resolve to
// tx.sender) survives a JSON round-trip distinctly from an explicit zero
// address, since the two select different signing keys.
func TestFrameTxJSONSignerRoundtrip(t *testing.T) {
	tx := sampleFrameTx(t)
	tx.Signatures = []FrameSignature{
		{Scheme: SignatureSchemeSecp256k1, Signer: nil, Signature: make([]byte, 65)},
		{Scheme: SignatureSchemeSecp256k1, Signer: make([]byte, 20), Signature: make([]byte, 65)},
	}
	enc, err := NewTx(tx).MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var dec Transaction
	if err := dec.UnmarshalJSON(enc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := dec.Inner().(*FrameTx).Signatures
	if len(got[0].Signer) != 0 {
		t.Errorf("absent signer became %x", got[0].Signer)
	}
	if len(got[1].Signer) != 20 {
		t.Errorf("explicit zero signer became %x, want 20 zero bytes", got[1].Signer)
	}
}

// TestFrameReceiptStorageRoundtrip checks that a frame receipt survives the
// storage encoding, and that it does not take the rest of the block's receipts
// down with it. The storage form carries no type byte, so it has to be
// self-describing: decoding happens before the type is known.
func TestFrameReceiptStorageRoundtrip(t *testing.T) {
	logA := &Log{Address: common.Address{0x11}, Topics: []common.Hash{{0x01}}, Data: []byte{0xaa}}
	logB := &Log{Address: common.Address{0x22}, Topics: []common.Hash{{0x02}}, Data: []byte{0xbb}}
	frameReceipt := &Receipt{
		Type:              FrameTxType,
		Status:            ReceiptStatusSuccessful,
		CumulativeGasUsed: 42_000,
		Payer:             common.Address{0x99},
		FrameReceipts: []*FrameReceipt{
			{Status: ReceiptStatusSuccessful, GasUsed: 1000, Logs: []*Log{logA}},
			{Status: ReceiptStatusFailed, GasUsed: 2000},
			{Status: ReceiptStatusSkipped},
			{Status: ReceiptStatusSuccessful, GasUsed: 3000, Logs: []*Log{logB}},
		},
	}
	legacyReceipt := &Receipt{
		Type:              LegacyTxType,
		Status:            ReceiptStatusSuccessful,
		CumulativeGasUsed: 21_000,
		Logs:              []*Log{logA},
	}
	// Encode a whole block's receipts, mixing a legacy receipt with the frame
	// one: a decode failure on either used to nil out the entire list.
	stored := []*ReceiptForStorage{(*ReceiptForStorage)(legacyReceipt), (*ReceiptForStorage)(frameReceipt)}
	enc, err := rlp.EncodeToBytes(stored)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var dec []*ReceiptForStorage
	if err := rlp.DecodeBytes(enc, &dec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(dec) != 2 {
		t.Fatalf("got %d receipts, want 2", len(dec))
	}
	// The legacy receipt must be untouched by the new optional fields.
	if got := (*Receipt)(dec[0]); got.Payer != (common.Address{}) || got.FrameReceipts != nil {
		t.Errorf("legacy receipt gained frame fields: payer %v, frames %v", got.Payer, got.FrameReceipts)
	}
	got := (*Receipt)(dec[1])
	if got.Payer != frameReceipt.Payer {
		t.Errorf("payer = %v, want %v", got.Payer, frameReceipt.Payer)
	}
	if len(got.FrameReceipts) != len(frameReceipt.FrameReceipts) {
		t.Fatalf("got %d frame receipts, want %d", len(got.FrameReceipts), len(frameReceipt.FrameReceipts))
	}
	for i, want := range frameReceipt.FrameReceipts {
		if got.FrameReceipts[i].Status != want.Status || got.FrameReceipts[i].GasUsed != want.GasUsed {
			t.Errorf("frame %d = {%d, %d}, want {%d, %d}", i,
				got.FrameReceipts[i].Status, got.FrameReceipts[i].GasUsed, want.Status, want.GasUsed)
		}
	}
	// Receipt.Logs must be rebuilt as the in-order concatenation of the frame
	// logs, otherwise DeriveFields assigns no log indices and eth_getLogs is blind.
	if len(got.Logs) != 2 || got.Logs[0].Address != logA.Address || got.Logs[1].Address != logB.Address {
		t.Errorf("logs = %v, want the concatenation of the frame logs", got.Logs)
	}
}

// TestNonFrameReceiptStorageUnchanged checks that a non-frame receipt still
// encodes as a three-element list, i.e. the trailing optional frame fields are
// omitted and existing stored receipts stay readable.
func TestNonFrameReceiptStorageUnchanged(t *testing.T) {
	r := &Receipt{
		Type:              DynamicFeeTxType,
		Status:            ReceiptStatusSuccessful,
		CumulativeGasUsed: 21_000,
		Logs:              []*Log{{Address: common.Address{0x11}}},
	}
	enc, err := rlp.EncodeToBytes((*ReceiptForStorage)(r))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var fields []rlp.RawValue
	if err := rlp.DecodeBytes(enc, &fields); err != nil {
		t.Fatalf("decode as list: %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("stored receipt has %d fields, want 3 (status, cumulativeGasUsed, logs)", len(fields))
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
