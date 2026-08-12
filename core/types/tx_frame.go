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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256r1"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// EIP-8141 frame transaction constants.
const (
	// Frame modes.
	FrameModeDefault byte = 0 // Execute the frame as ENTRY_POINT
	FrameModeVerify  byte = 1 // Frame identifies as transaction validation
	FrameModeSender  byte = 2 // Execute the frame as tx.sender

	// Frame flags.
	FrameFlagApprovePayment          = byte(0x1) // frame may approve payment
	FrameFlagApproveExecution        = byte(0x2) // frame may approve execution
	FrameFlagApproveExecutionPayment = byte(0x3) // mask of both approval scopes
	FrameFlagAtomicBatch             = byte(0x4) // frame is part of an atomic batch

	// Signature schemes.
	SignatureSchemeArbitrary byte = 0x0
	SignatureSchemeSecp256k1 byte = 0x1
	SignatureSchemeP256      byte = 0x2
)

// Frame represents a single frame within a frame transaction.
type Frame struct {
	Mode     byte
	Flags    byte
	Target   *common.Address `rlp:"nil"` // nil means tx.sender
	GasLimit uint64
	Value    *uint256.Int
	Data     []byte
}

// Value256 returns the frame's value, treating a nil value as zero. A frame
// decoded from RLP always has a non-nil value, but one built in memory may not.
func (f *Frame) Value256() *uint256.Int {
	if f.Value == nil {
		return new(uint256.Int)
	}
	return f.Value
}

// copy returns a deep copy of the frame.
func (f *Frame) copy() *Frame {
	cpy := &Frame{
		Mode:     f.Mode,
		Flags:    f.Flags,
		Target:   copyAddressPtr(f.Target),
		GasLimit: f.GasLimit,
		Value:    new(uint256.Int),
		Data:     common.CopyBytes(f.Data),
	}
	if f.Value != nil {
		cpy.Value.Set(f.Value)
	}
	return cpy
}

// FrameSignature is a signature entry within a frame transaction.
type FrameSignature struct {
	Scheme    byte
	Signer    []byte // empty or 20-byte address
	Msg       []byte // empty or 32-byte digest
	Signature []byte // raw signature bytes
}

// copy returns a deep copy of the signature entry.
func (s *FrameSignature) copy() *FrameSignature {
	return &FrameSignature{
		Scheme:    s.Scheme,
		Signer:    common.CopyBytes(s.Signer),
		Msg:       common.CopyBytes(s.Msg),
		Signature: common.CopyBytes(s.Signature),
	}
}

// ResolvedSigner returns the signer address, defaulting to the transaction
// sender when no explicit signer is provided.
func (s *FrameSignature) ResolvedSigner(sender common.Address) (common.Address, bool) {
	if len(s.Signer) == 0 {
		return sender, true
	}
	if len(s.Signer) == 20 {
		return common.BytesToAddress(s.Signer), true
	}
	return common.Address{}, false
}

// FrameTx represents an EIP-8141 frame transaction.
type FrameTx struct {
	ChainID    *big.Int
	Nonce      uint64
	Sender     common.Address
	Frames     []Frame
	Signatures []FrameSignature

	GasTipCap  *big.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *big.Int // a.k.a. maxFeePerGas
	BlobFeeCap *uint256.Int
	BlobHashes []common.Hash

	// Signature values: frame transactions carry no outer ECDSA signature.
	// These fields are excluded from the RLP payload and retained only to
	// satisfy the TxData interface.
	V *big.Int `rlp:"-"`
	R *big.Int `rlp:"-"`
	S *big.Int `rlp:"-"`
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *FrameTx) copy() TxData {
	cpy := &FrameTx{
		Nonce:      tx.Nonce,
		Sender:     tx.Sender,
		Frames:     make([]Frame, len(tx.Frames)),
		BlobHashes: make([]common.Hash, len(tx.BlobHashes)),
		ChainID:    new(big.Int),
		GasTipCap:  new(big.Int),
		GasFeeCap:  new(big.Int),
		BlobFeeCap: new(uint256.Int),
		V:          new(big.Int),
		R:          new(big.Int),
		S:          new(big.Int),
	}
	for i := range tx.Frames {
		cpy.Frames[i] = *tx.Frames[i].copy()
	}
	cpy.Signatures = make([]FrameSignature, len(tx.Signatures))
	for i := range tx.Signatures {
		cpy.Signatures[i] = *tx.Signatures[i].copy()
	}
	copy(cpy.BlobHashes, tx.BlobHashes)
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	if tx.BlobFeeCap != nil {
		cpy.BlobFeeCap.Set(tx.BlobFeeCap)
	}
	return cpy
}

// accessors for innerTx.
func (tx *FrameTx) txType() byte           { return FrameTxType }
func (tx *FrameTx) chainID() *big.Int      { return tx.ChainID }
func (tx *FrameTx) accessList() AccessList { return nil }
func (tx *FrameTx) data() []byte           { return nil }
func (tx *FrameTx) gasFeeCap() *big.Int    { return tx.GasFeeCap }
func (tx *FrameTx) gasTipCap() *big.Int    { return tx.GasTipCap }
func (tx *FrameTx) gasPrice() *big.Int     { return tx.GasFeeCap }
func (tx *FrameTx) value() *big.Int        { return new(big.Int) }
func (tx *FrameTx) nonce() uint64          { return tx.Nonce }
func (tx *FrameTx) to() *common.Address    { return nil }

// gas returns the maximum gas the transaction is allowed to consume (EIP-8141
// max_gas). Frame transactions have no explicit gas field; it is derived from
// the frames and signatures.
func (tx *FrameTx) gas() uint64 {
	return tx.MaxGas()
}

func (tx *FrameTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap)
	}
	tip := dst.Sub(tx.GasFeeCap, baseFee)
	if tip.Cmp(tx.GasTipCap) > 0 {
		tip.Set(tx.GasTipCap)
	}
	return tip.Add(tip, baseFee)
}

func (tx *FrameTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *FrameTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}

func (tx *FrameTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *FrameTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}

// sigHash returns the canonical transaction signature hash. The raw signature
// bytes of any signature entry with empty msg are elided before hashing.
func (tx *FrameTx) sigHash(chainID *big.Int) common.Hash {
	return tx.ComputeSigHash()
}

// ComputeSigHash computes the canonical signature hash of the frame
// transaction. Signatures with empty msg have their raw signature bytes elided
// from the encoded payload.
func (tx *FrameTx) ComputeSigHash() common.Hash {
	cpy := tx.copy().(*FrameTx)
	for i := range cpy.Signatures {
		if len(cpy.Signatures[i].Msg) == 0 {
			cpy.Signatures[i].Signature = nil
		}
	}
	return prefixedRlpHash(FrameTxType, cpy)
}

// SignatureVerificationCost returns the total gas cost of verifying all
// protocol-validated signature entries (EIP-8141 signature_gas).
func (tx *FrameTx) SignatureVerificationCost() (uint64, error) {
	var total uint64
	for i := range tx.Signatures {
		var cost uint64
		switch tx.Signatures[i].Scheme {
		case SignatureSchemeSecp256k1:
			cost = params.FrameTxSignatureSecp256k1
		case SignatureSchemeP256:
			cost = params.FrameTxSignatureP256
		case SignatureSchemeArbitrary:
			cost = params.FrameTxSignatureArbitrary
		default:
			return 0, errors.New("invalid signature scheme")
		}
		if math.MaxUint64-total < cost {
			return 0, errors.New("signature verification cost overflow")
		}
		total += cost
	}
	return total, nil
}

// ErrFrameGasOverflow is returned when a frame transaction's declared gas
// figures do not fit in 64 bits, which makes the transaction invalid.
var ErrFrameGasOverflow = errors.New("frame transaction gas overflow")

func addGas(a, b uint64) (uint64, error) {
	if math.MaxUint64-a < b {
		return 0, ErrFrameGasOverflow
	}
	return a + b, nil
}

func mulGas(a, b uint64) (uint64, error) {
	if a != 0 && math.MaxUint64/a < b {
		return 0, ErrFrameGasOverflow
	}
	return a * b, nil
}

// tokenIn returns the number of data tokens in b (EIP-8141 tokens_in): one
// token per zero byte and TxTokenPerNonZeroByte per non-zero byte.
func tokenIn(b []byte) uint64 {
	var zero uint64
	for _, c := range b {
		if c == 0 {
			zero++
		}
	}
	// No overflow is possible: len(b) is bounded by the decoded payload size.
	return zero + (uint64(len(b))-zero)*params.TxTokenPerNonZeroByte
}

// calldataTokens returns the total number of data tokens across frames and
// signatures (EIP-8141 calldata_tokens).
func (tx *FrameTx) calldataTokens() (uint64, error) {
	var (
		tokens uint64
		err    error
	)
	for i := range tx.Frames {
		if tokens, err = addGas(tokens, tokenIn(tx.Frames[i].Data)); err != nil {
			return 0, err
		}
	}
	for i := range tx.Signatures {
		s := &tx.Signatures[i]
		for _, b := range [][]byte{s.Signer, s.Msg, s.Signature} {
			if tokens, err = addGas(tokens, tokenIn(b)); err != nil {
				return 0, err
			}
		}
	}
	return tokens, nil
}

// SumFrameGas returns the sum of all frame gas limits.
func (tx *FrameTx) SumFrameGas() (uint64, error) {
	var (
		total uint64
		err   error
	)
	for i := range tx.Frames {
		if total, err = addGas(total, tx.Frames[i].GasLimit); err != nil {
			return 0, err
		}
	}
	return total, nil
}

// GasLimits computes the three EIP-8141 gas figures of the transaction:
// standard_gas_limit, calldata_floor_gas and max_gas.
//
// Both limits share a base of the intrinsic cost, the per-frame cost and the
// signature verification cost. The standard limit adds the calldata cost at
// STANDARD_TOKEN_COST per token plus the sum of the frame gas limits; the floor
// charges TOTAL_COST_FLOOR_PER_TOKEN per token instead and excludes frame gas.
func (tx *FrameTx) GasLimits() (standard, floor, maxGas uint64, err error) {
	sigVerify, err := tx.SignatureVerificationCost()
	if err != nil {
		return 0, 0, 0, err
	}
	base, err := mulGas(uint64(len(tx.Frames)), params.FrameTxPerFrameCost)
	if err != nil {
		return 0, 0, 0, err
	}
	if base, err = addGas(base, params.FrameTxIntrinsicCost); err != nil {
		return 0, 0, 0, err
	}
	if base, err = addGas(base, sigVerify); err != nil {
		return 0, 0, 0, err
	}
	tokens, err := tx.calldataTokens()
	if err != nil {
		return 0, 0, 0, err
	}
	// standard_gas_limit = base + STANDARD_TOKEN_COST*tokens + sum(frame.gas_limit).
	// STANDARD_TOKEN_COST is 4, which params spells as the per-zero-byte cost.
	calldataCost, err := mulGas(tokens, params.TxDataZeroGas)
	if err != nil {
		return 0, 0, 0, err
	}
	if standard, err = addGas(base, calldataCost); err != nil {
		return 0, 0, 0, err
	}
	frameGas, err := tx.SumFrameGas()
	if err != nil {
		return 0, 0, 0, err
	}
	if standard, err = addGas(standard, frameGas); err != nil {
		return 0, 0, 0, err
	}
	// calldata_floor_gas = base + TOTAL_COST_FLOOR_PER_TOKEN*tokens.
	floorCost, err := mulGas(tokens, params.TxCostFloorPerToken)
	if err != nil {
		return 0, 0, 0, err
	}
	if floor, err = addGas(base, floorCost); err != nil {
		return 0, 0, 0, err
	}
	return standard, floor, max(standard, floor), nil
}

// MaxGas returns the maximum gas the transaction may consume (EIP-8141 max_gas),
// saturating to MaxUint64 when the figures overflow. Overflow makes the
// transaction invalid, which validateFrameTx reports via GasLimits; saturating
// here keeps the value monotonic so the block gas check rejects it either way.
func (tx *FrameTx) MaxGas() uint64 {
	_, _, maxGas, err := tx.GasLimits()
	if err != nil {
		return math.MaxUint64
	}
	return maxGas
}

// ValidateSignature validates a single signature entry against the canonical
// transaction signature hash (EIP-8141 validate_signature). It reports whether
// the signature is structurally valid and, for protocol-validated schemes,
// cryptographically valid.
func (tx *FrameTx) ValidateSignature(sig *FrameSignature, sigHash common.Hash) bool {
	var msg common.Hash
	switch len(sig.Msg) {
	case 0:
		msg = sigHash
	case 32:
		if isZeroHash(sig.Msg) {
			return false
		}
		copy(msg[:], sig.Msg)
	default:
		return false
	}
	resolved, ok := sig.ResolvedSigner(tx.Sender)
	if !ok {
		return false
	}
	switch sig.Scheme {
	case SignatureSchemeSecp256k1:
		if len(sig.Signature) != 65 {
			return false
		}
		v := sig.Signature[0]
		r := new(big.Int).SetBytes(sig.Signature[1:33])
		s := new(big.Int).SetBytes(sig.Signature[33:65])
		// v is the recovery id (0 or 1); r and s must be canonical with low-s.
		if v > 1 || !crypto.ValidateSignatureValues(v, r, s, true) {
			return false
		}
		// Ecrecover expects [R||S||V]; the frame encoding is [V||R||S].
		canon := make([]byte, 65)
		copy(canon[:32], sig.Signature[1:33])
		copy(canon[32:64], sig.Signature[33:65])
		canon[64] = v
		recovered, err := crypto.Ecrecover(msg[:], canon)
		if err != nil {
			return false
		}
		pub, err := crypto.UnmarshalPubkey(recovered)
		if err != nil {
			return false
		}
		return crypto.PubkeyToAddress(*pub) == resolved
	case SignatureSchemeP256:
		return validateP256Signature(sig, msg, resolved)
	case SignatureSchemeArbitrary:
		return len(sig.Signer) == 0
	default:
		return false
	}
}

// isZeroHash reports whether b is the 32-byte zero digest.
func isZeroHash(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

// validateP256Signature validates an EIP-8141 P256 signature entry using the
// P256 verifier (EIP-7212 style P256VERIFY).
func validateP256Signature(sig *FrameSignature, msg common.Hash, resolved common.Address) bool {
	if len(sig.Signature) != 128 {
		return false
	}
	// r, s, qx, qy are 32 bytes each. Reject zero and high-s per the spec.
	r := new(big.Int).SetBytes(sig.Signature[0:32])
	s := new(big.Int).SetBytes(sig.Signature[32:64])
	secp256r1N, _ := new(big.Int).SetString("ffffffff00000000ffffffffffffffffbce6faada7179e84f3b9cac2fc632551", 16)
	if r.Sign() <= 0 || s.Sign() <= 0 || r.Cmp(secp256r1N) >= 0 || s.Cmp(new(big.Int).Rsh(secp256r1N, 1)) > 0 {
		return false
	}
	qx := new(big.Int).SetBytes(sig.Signature[64:96])
	qy := new(big.Int).SetBytes(sig.Signature[96:128])
	if qx.Sign() == 0 && qy.Sign() == 0 {
		return false
	}
	// The signer address must be keccak256(qx || qy)[12:].
	addr := common.BytesToAddress(crypto.Keccak256(sig.Signature[64:96], sig.Signature[96:128])[12:])
	if addr != resolved {
		return false
	}
	return secp256r1.Verify(msg[:], r, s, qx, qy)
}
