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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256r1"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

var (
	ErrFrameTxInvalidFormat    = errors.New("invalid frame tx format")
	ErrFrameTxInvalidSignature = errors.New("invalid frame tx signature")
)

// Frame execution modes (EIP-8141).
const (
	FrameTxModeDefault uint64 = 0
	FrameTxModeVerify  uint64 = 1
	FrameTxModeSender  uint64 = 2
)

// Frame flags (EIP-8141). Bits 0-1 hold the allowed approval scope, bit 2
// marks the frame as part of an atomic batch. Higher bits are reserved.
const (
	FrameTxApproveNone                uint64 = 0x0
	FrameTxApprovePayment             uint64 = 0x1
	FrameTxApproveExecution           uint64 = 0x2
	FrameTxApproveExecutionAndPayment uint64 = 0x3
	FrameTxApproveScopeMask           uint64 = 0x3
	FrameTxAtomicBatchFlag            uint64 = 0x4
	FrameTxFlagsLimit                 uint64 = 0x8
)

// Signature schemes (EIP-8141).
const (
	FrameTxSchemeArbitrary uint64 = 0x0
	FrameTxSchemeSecp256k1 uint64 = 0x1
	FrameTxSchemeP256      uint64 = 0x2
)

// FrameTxFrame is a single frame in a frame transaction.
type FrameTxFrame struct {
	Mode     uint64
	Flags    uint64
	Target   *common.Address `rlp:"nil"` // nil resolves to the transaction sender
	GasLimit uint64
	Value    *uint256.Int
	Data     []byte
}

// ResolvedTarget returns the frame's target, resolving a nil target to the
// transaction sender.
func (f *FrameTxFrame) ResolvedTarget(sender common.Address) common.Address {
	if f.Target == nil {
		return sender
	}
	return *f.Target
}

// IsExpiryVerifier reports whether the frame is an expiry verifier frame,
// i.e. a VERIFY frame targeting the expiry verifier contract.
func (f *FrameTxFrame) IsExpiryVerifier() bool {
	return f.Mode == FrameTxModeVerify && f.Target != nil && *f.Target == params.FrameTxExpiryVerifier
}

type frameTxFrameJSON struct {
	Mode     hexutil.Uint64  `json:"mode"`
	Flags    hexutil.Uint64  `json:"flags"`
	Target   *common.Address `json:"target,omitempty"`
	GasLimit hexutil.Uint64  `json:"gasLimit"`
	Value    *hexutil.U256   `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
}

func (f FrameTxFrame) MarshalJSON() ([]byte, error) {
	enc := frameTxFrameJSON{
		Mode:     hexutil.Uint64(f.Mode),
		Flags:    hexutil.Uint64(f.Flags),
		Target:   f.Target,
		GasLimit: hexutil.Uint64(f.GasLimit),
		Value:    (*hexutil.U256)(f.Value),
		Data:     f.Data,
	}
	return json.Marshal(&enc)
}

func (f *FrameTxFrame) UnmarshalJSON(input []byte) error {
	var dec frameTxFrameJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	f.Mode = uint64(dec.Mode)
	f.Flags = uint64(dec.Flags)
	f.Target = dec.Target
	f.GasLimit = uint64(dec.GasLimit)
	f.Value = new(uint256.Int)
	if dec.Value != nil {
		f.Value = (*uint256.Int)(dec.Value)
	}
	f.Data = dec.Data
	return nil
}

// FrameTxSignature is a signature entry in a frame transaction.
type FrameTxSignature struct {
	Scheme    uint64
	Signer    []byte // 20-byte address for SECP256K1 and P256, empty for ARBITRARY
	Msg       []byte // empty (canonical signature hash) or an explicit 32-byte digest
	Signature []byte
}

type frameTxSignatureJSON struct {
	Scheme    hexutil.Uint64 `json:"scheme"`
	Signer    hexutil.Bytes  `json:"signer"`
	Msg       hexutil.Bytes  `json:"msg"`
	Signature hexutil.Bytes  `json:"signature"`
}

func (s FrameTxSignature) MarshalJSON() ([]byte, error) {
	enc := frameTxSignatureJSON{
		Scheme:    hexutil.Uint64(s.Scheme),
		Signer:    s.Signer,
		Msg:       s.Msg,
		Signature: s.Signature,
	}
	return json.Marshal(&enc)
}

func (s *FrameTxSignature) UnmarshalJSON(input []byte) error {
	var dec frameTxSignatureJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	s.Scheme = uint64(dec.Scheme)
	s.Signer = dec.Signer
	s.Msg = dec.Msg
	s.Signature = dec.Signature
	return nil
}

// FrameTx represents an EIP-8141 frame transaction.
type FrameTx struct {
	ChainID    *uint256.Int
	Nonce      uint64
	Sender     common.Address
	Frames     []FrameTxFrame
	Signatures []FrameTxSignature

	MaxPriorityFeePerGas *uint256.Int
	MaxFeePerGas         *uint256.Int
	MaxFeePerBlobGas     *uint256.Int
	BlobVersionedHashes  []common.Hash
}

func (tx *FrameTx) copy() TxData {
	cpy := &FrameTx{
		Nonce:               tx.Nonce,
		Sender:              tx.Sender,
		Frames:              make([]FrameTxFrame, len(tx.Frames)),
		Signatures:          make([]FrameTxSignature, len(tx.Signatures)),
		BlobVersionedHashes: make([]common.Hash, len(tx.BlobVersionedHashes)),

		ChainID:              new(uint256.Int),
		MaxPriorityFeePerGas: new(uint256.Int),
		MaxFeePerGas:         new(uint256.Int),
		MaxFeePerBlobGas:     new(uint256.Int),
	}
	for i, frame := range tx.Frames {
		var target *common.Address
		if frame.Target != nil {
			t := *frame.Target
			target = &t
		}
		value := new(uint256.Int)
		if frame.Value != nil {
			value.Set(frame.Value)
		}
		cpy.Frames[i] = FrameTxFrame{
			Mode:     frame.Mode,
			Flags:    frame.Flags,
			Target:   target,
			GasLimit: frame.GasLimit,
			Value:    value,
			Data:     common.CopyBytes(frame.Data),
		}
	}
	for i, sig := range tx.Signatures {
		cpy.Signatures[i] = FrameTxSignature{
			Scheme:    sig.Scheme,
			Signer:    common.CopyBytes(sig.Signer),
			Msg:       common.CopyBytes(sig.Msg),
			Signature: common.CopyBytes(sig.Signature),
		}
	}
	copy(cpy.BlobVersionedHashes, tx.BlobVersionedHashes)
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.MaxPriorityFeePerGas != nil {
		cpy.MaxPriorityFeePerGas.Set(tx.MaxPriorityFeePerGas)
	}
	if tx.MaxFeePerGas != nil {
		cpy.MaxFeePerGas.Set(tx.MaxFeePerGas)
	}
	if tx.MaxFeePerBlobGas != nil {
		cpy.MaxFeePerBlobGas.Set(tx.MaxFeePerBlobGas)
	}
	return cpy
}

func (tx *FrameTx) txType() byte { return FrameTxType }
func (tx *FrameTx) chainID() *big.Int {
	if tx.ChainID == nil {
		return new(big.Int)
	}
	return tx.ChainID.ToBig()
}
func (tx *FrameTx) nonce() uint64          { return tx.Nonce }
func (tx *FrameTx) to() *common.Address    { return nil }
func (tx *FrameTx) value() *big.Int        { return common.Big0 }
func (tx *FrameTx) data() []byte           { return nil }
func (tx *FrameTx) accessList() AccessList { return nil }
func (tx *FrameTx) gasFeeCap() *big.Int {
	if tx.MaxFeePerGas == nil {
		return new(big.Int)
	}
	return tx.MaxFeePerGas.ToBig()
}
func (tx *FrameTx) gasTipCap() *big.Int {
	if tx.MaxPriorityFeePerGas == nil {
		return new(big.Int)
	}
	return tx.MaxPriorityFeePerGas.ToBig()
}
func (tx *FrameTx) gasPrice() *big.Int {
	if tx.MaxFeePerGas == nil {
		return new(big.Int)
	}
	return tx.MaxFeePerGas.ToBig()
}

// gas returns the derived total gas limit of the frame transaction.
func (tx *FrameTx) gas() uint64 {
	total, err := FrameTxGas(tx.Frames, tx.Signatures)
	if err != nil {
		return 0
	}
	return total
}

func (tx *FrameTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.gasFeeCap())
	}
	tip := dst.Sub(tx.gasFeeCap(), baseFee)
	if tip.Cmp(tx.gasTipCap()) > 0 {
		tip.Set(tx.gasTipCap())
	}
	return tip.Add(tip, baseFee)
}

func (tx *FrameTx) rawSignatureValues() (v, r, s *big.Int) {
	return nil, nil, nil
}

func (tx *FrameTx) setSignatureValues(chainID, v, r, s *big.Int) {}

func (tx *FrameTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *FrameTx) decode(input []byte) error {
	if err := rlp.DecodeBytes(input, tx); err != nil {
		return fmt.Errorf("%w: %v", ErrFrameTxInvalidFormat, err)
	}
	return tx.validate()
}

// validate checks the static constraints of the frame transaction defined
// in EIP-8141.
func (tx *FrameTx) validate() error {
	if tx.ChainID == nil || tx.MaxPriorityFeePerGas == nil || tx.MaxFeePerGas == nil || tx.MaxFeePerBlobGas == nil {
		return ErrFrameTxInvalidFormat
	}
	if len(tx.Frames) == 0 || len(tx.Frames) > params.FrameTxMaxFrames {
		return fmt.Errorf("%w: frame count must be greater than 0 and at most %d", ErrFrameTxInvalidFormat, params.FrameTxMaxFrames)
	}
	for _, sig := range tx.Signatures {
		switch sig.Scheme {
		case FrameTxSchemeSecp256k1, FrameTxSchemeP256:
			if len(sig.Signer) != 0 && len(sig.Signer) != common.AddressLength {
				return fmt.Errorf("%w: signer must be empty or a 20-byte address", ErrFrameTxInvalidFormat)
			}
		case FrameTxSchemeArbitrary:
			if len(sig.Signer) != 0 {
				return fmt.Errorf("%w: arbitrary signature signer must be empty", ErrFrameTxInvalidFormat)
			}
		default:
			return fmt.Errorf("%w: unknown signature scheme", ErrFrameTxInvalidFormat)
		}
		switch len(sig.Msg) {
		case 0:
		case common.HashLength:
			if common.Hash(sig.Msg) == (common.Hash{}) {
				return fmt.Errorf("%w: explicit zero digest is invalid", ErrFrameTxInvalidFormat)
			}
		default:
			return fmt.Errorf("%w: signature msg must be empty or 32 bytes", ErrFrameTxInvalidFormat)
		}
	}
	var (
		totalFrameGas        uint64
		expiryVerifierFrames int
	)
	for i, frame := range tx.Frames {
		if frame.Mode >= 3 {
			return fmt.Errorf("%w: unknown frame mode", ErrFrameTxInvalidFormat)
		}
		if frame.Flags >= FrameTxFlagsLimit {
			return fmt.Errorf("%w: reserved frame flags set", ErrFrameTxInvalidFormat)
		}
		if frame.Value == nil {
			return ErrFrameTxInvalidFormat
		}
		if frame.Mode != FrameTxModeSender && !frame.Value.IsZero() {
			return fmt.Errorf("%w: non-zero value outside SENDER mode", ErrFrameTxInvalidFormat)
		}
		if math.MaxUint64-totalFrameGas < frame.GasLimit {
			return fmt.Errorf("%w: total frame gas too high", ErrFrameTxInvalidFormat)
		}
		totalFrameGas += frame.GasLimit

		// Execution approval is only allowed for frames that resolve to
		// the transaction sender.
		if frame.Flags&FrameTxApproveExecution != 0 && frame.ResolvedTarget(tx.Sender) != tx.Sender {
			return fmt.Errorf("%w: execution approval flag outside sender target", ErrFrameTxInvalidFormat)
		}

		// An atomic batch must be terminated by a subsequent frame.
		if frame.Flags&FrameTxAtomicBatchFlag != 0 && i+1 >= len(tx.Frames) {
			return fmt.Errorf("%w: atomic batch flag set on last frame", ErrFrameTxInvalidFormat)
		}

		if frame.IsExpiryVerifier() {
			expiryVerifierFrames++
			if frame.Flags != 0 {
				return fmt.Errorf("%w: expiry verifier frame flags must be zero", ErrFrameTxInvalidFormat)
			}
			if len(frame.Data) != params.FrameTxExpiryDataLen {
				return fmt.Errorf("%w: expiry verifier frame data must be 8 bytes", ErrFrameTxInvalidFormat)
			}
		}
	}
	if expiryVerifierFrames > 1 {
		return fmt.Errorf("%w: multiple expiry verifier frames", ErrFrameTxInvalidFormat)
	}
	if len(tx.BlobVersionedHashes) == 0 && !tx.MaxFeePerBlobGas.IsZero() {
		return fmt.Errorf("%w: max fee per blob gas must be zero without blobs", ErrFrameTxInvalidFormat)
	}
	if tx.Nonce == math.MaxUint64 {
		return fmt.Errorf("%w: nonce overflow", ErrFrameTxInvalidFormat)
	}
	if _, err := FrameTxGas(tx.Frames, tx.Signatures); err != nil {
		return fmt.Errorf("%w: %v", ErrFrameTxInvalidFormat, err)
	}
	totalGas, _ := FrameTxGas(tx.Frames, tx.Signatures)
	floorGas, err := FrameTxFloorGas(tx.Frames, tx.Signatures)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFrameTxInvalidFormat, err)
	}
	if floorGas > totalGas {
		return fmt.Errorf("%w: insufficient calldata floor: have %d, want %d", ErrFrameTxInvalidFormat, totalGas, floorGas)
	}
	return nil
}

// sigHash returns the canonical signature hash of the frame transaction.
// The raw signature bytes of every entry with an empty msg are elided.
func (tx *FrameTx) sigHash(chainID *big.Int) common.Hash {
	if chainID == nil {
		chainID = tx.chainID()
	}
	sigs := make([]FrameTxSignature, len(tx.Signatures))
	for i, sig := range tx.Signatures {
		sigs[i] = sig
		if len(sig.Msg) == 0 {
			sigs[i].Signature = nil
		}
	}
	return prefixedRlpHash(
		FrameTxType,
		[]any{
			chainID,
			tx.Nonce,
			tx.Sender,
			tx.Frames,
			sigs,
			tx.MaxPriorityFeePerGas,
			tx.MaxFeePerGas,
			tx.MaxFeePerBlobGas,
			tx.BlobVersionedHashes,
		},
	)
}

// FrameTxSignatureGas returns the gas charged for the protocol validation of
// a single signature entry.
func FrameTxSignatureGas(sig *FrameTxSignature) uint64 {
	switch sig.Scheme {
	case FrameTxSchemeSecp256k1:
		return params.FrameTxSecp256k1SigGas
	case FrameTxSchemeP256:
		return params.FrameTxP256SigGas
	default:
		return params.FrameTxArbitrarySigGas
	}
}

// FrameTxChargedData returns the byte fields of a frame transaction that are
// priced as calldata: the data of each frame and the signer, msg and
// signature bytes of each signature entry. The fixed-size fields are covered
// by the intrinsic and per-frame costs.
func FrameTxChargedData(frames []FrameTxFrame, sigs []FrameTxSignature) [][]byte {
	charged := make([][]byte, 0, len(frames)+3*len(sigs))
	for i := range frames {
		charged = append(charged, frames[i].Data)
	}
	for i := range sigs {
		charged = append(charged, sigs[i].Signer, sigs[i].Msg, sigs[i].Signature)
	}
	return charged
}

// FrameTxGas calculates the derived total gas limit of a frame transaction
// as specified by EIP-8141: the frame transaction intrinsic cost, the
// per-frame cost, the calldata cost of the frame and signature byte fields,
// the signature verification cost, and the gas limits of all frames.
func FrameTxGas(frames []FrameTxFrame, sigs []FrameTxSignature) (uint64, error) {
	chargedData := FrameTxChargedData(frames, sigs)
	total := params.FrameTxIntrinsicGas
	total += uint64(len(frames)) * params.FrameTxPerFrameGas
	for _, data := range chargedData {
		z := uint64(bytes.Count(data, []byte{0}))
		nz := uint64(len(data)) - z
		total += z*params.TxDataZeroGas + nz*params.TxDataNonZeroGasEIP2028
	}
	for i := range sigs {
		total += FrameTxSignatureGas(&sigs[i])
	}
	for _, frame := range frames {
		if math.MaxUint64-total < frame.GasLimit {
			return 0, errors.New("gas uint64 overflow")
		}
		total += frame.GasLimit
	}
	return total, nil
}

// FrameTxFloorGas computes the minimum gas cost of a frame transaction based
// on the size of the frame and signature byte fields, per EIP-7623 and
// EIP-7976: every charged byte counts as a standard token priced at the
// floor token cost.
func FrameTxFloorGas(frames []FrameTxFrame, sigs []FrameTxSignature) (uint64, error) {
	chargedData := FrameTxChargedData(frames, sigs)
	var dataLen uint64
	for _, data := range chargedData {
		dataLen += uint64(len(data))
	}
	if math.MaxUint64/(params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken7976) < dataLen {
		return 0, errors.New("gas uint64 overflow")
	}
	return params.FrameTxIntrinsicGas + dataLen*params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken7976, nil
}

// FrameTxMaxCost returns the maximum cost of the frame transaction that is
// collected from the payer upon payment approval: the total gas limit priced
// at the max fee per gas plus the blob fees priced at the max fee per blob
// gas.
func (tx *FrameTx) FrameTxMaxCost() *uint256.Int {
	total, err := FrameTxGas(tx.Frames, tx.Signatures)
	if err != nil {
		return new(uint256.Int)
	}
	cost := new(uint256.Int).SetUint64(total)
	cost.Mul(cost, tx.MaxFeePerGas)
	if n := len(tx.BlobVersionedHashes); n > 0 {
		blobFee := new(uint256.Int).SetUint64(uint64(n) * params.BlobTxBlobGasPerBlob)
		blobFee.Mul(blobFee, tx.MaxFeePerBlobGas)
		cost.Add(cost, blobFee)
	}
	return cost
}

// ValidateFrameTxSignatures validates all signature entries of a frame
// transaction against the canonical signature hash, per EIP-8141.
// ResolvedSigner returns the signer address of a protocol-validated
// signature entry, resolving an empty signer to the transaction sender.
func (s *FrameTxSignature) ResolvedSigner(sender common.Address) common.Address {
	if len(s.Signer) == 0 {
		return sender
	}
	return common.BytesToAddress(s.Signer)
}

func ValidateFrameTxSignatures(sigs []FrameTxSignature, sender common.Address, sigHash common.Hash) error {
	for i := range sigs {
		if !validateFrameTxSignature(&sigs[i], sender, sigHash) {
			return fmt.Errorf("%w: entry %d", ErrFrameTxInvalidSignature, i)
		}
	}
	return nil
}

func validateFrameTxSignature(sig *FrameTxSignature, sender common.Address, sigHash common.Hash) bool {
	var msg common.Hash
	switch len(sig.Msg) {
	case 0:
		msg = sigHash
	case common.HashLength:
		msg = common.Hash(sig.Msg)
		if msg == (common.Hash{}) {
			return false
		}
	default:
		return false
	}

	switch sig.Scheme {
	case FrameTxSchemeSecp256k1:
		if len(sig.Signature) != 65 {
			return false
		}
		v := sig.Signature[0]
		r := new(big.Int).SetBytes(sig.Signature[1:33])
		s := new(big.Int).SetBytes(sig.Signature[33:65])
		if !crypto.ValidateSignatureValues(v, r, s, true) {
			return false
		}
		// Ecrecover expects the signature as r || s || v.
		rsv := make([]byte, 65)
		copy(rsv, sig.Signature[1:65])
		rsv[64] = v
		pub, err := crypto.Ecrecover(msg[:], rsv)
		if err != nil {
			return false
		}
		var signer common.Address
		copy(signer[:], crypto.Keccak256(pub[1:])[12:])
		return sig.ResolvedSigner(sender) == signer

	case FrameTxSchemeP256:
		if len(sig.Signature) != 128 {
			return false
		}
		resolved := sig.ResolvedSigner(sender)
		if !bytes.Equal(resolved[:], crypto.Keccak256(sig.Signature[64:128])[12:]) {
			return false
		}
		r := new(big.Int).SetBytes(sig.Signature[0:32])
		s := new(big.Int).SetBytes(sig.Signature[32:64])
		x := new(big.Int).SetBytes(sig.Signature[64:96])
		y := new(big.Int).SetBytes(sig.Signature[96:128])
		return secp256r1.Verify(msg[:], r, s, x, y)

	case FrameTxSchemeArbitrary:
		return len(sig.Signer) == 0

	default:
		return false
	}
}
