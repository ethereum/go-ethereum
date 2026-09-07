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
	"crypto/elliptic"
	"encoding/json"
	"errors"
	"fmt"
	gomath "math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256r1"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

var (
	ErrFrameTxInvalidFormat    = errors.New("invalid frame tx format")
	ErrFrameTxInvalidSignature = errors.New("invalid frame tx signature")

	// secp256r1N and secp256r1HalfN bound canonical P-256 signature values
	// (EIP-8141): 0 < r < N and 0 < s <= N/2.
	secp256r1N     = elliptic.P256().Params().N
	secp256r1HalfN = new(big.Int).Rsh(elliptic.P256().Params().N, 1)
)

// Signature schemes (EIP-8141).
const (
	FrameTxSchemeArbitrary uint64 = 0x0
	FrameTxSchemeSecp256k1 uint64 = 0x1
	FrameTxSchemeP256      uint64 = 0x2
)

// FrameTx represents an EIP-8141 frame transaction.
type FrameTx struct {
	ChainID             *uint256.Int
	Nonce               uint64
	Sender              common.Address
	Frames              []Frame
	Signatures          SignatureList
	Fees                Fees
	BlobVersionedHashes []common.Hash
}

// FrameTxValidateStatic runs the EIP-8141 static-constraint checks if tx is
// a frame transaction, and reports nil for every other transaction type.
func (tx *Transaction) FrameTxValidateStatic() error {
	if ftx, ok := tx.inner.(*FrameTx); ok {
		return ftx.ValidateStatic()
	}
	return nil
}

// ValidateStatic checks the static constraints of the frame transaction
// defined in EIP-8141. The checks run at transaction validation time, not
// at decode time, so a structurally well-formed but statically invalid
// frame transaction decodes successfully and is rejected when applied.
func (tx *FrameTx) ValidateStatic() error {
	if tx.ChainID == nil || tx.Fees.MaxPriorityFeePerGas == nil || tx.Fees.MaxFeePerGas == nil || tx.Fees.MaxFeePerBlobGas == nil {
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
		if frame.Flags >= FlagsLimit {
			return fmt.Errorf("%w: reserved frame flags set", ErrFrameTxInvalidFormat)
		}
		if frame.Value == nil {
			return ErrFrameTxInvalidFormat
		}
		if frame.Mode != ModeSender && !frame.Value.IsZero() {
			return fmt.Errorf("%w: non-zero value outside SENDER mode", ErrFrameTxInvalidFormat)
		}
		// The frames' total gas budget, over both dimensions, must fit the
		// encoding limit.
		if gomath.MaxUint64-totalFrameGas < frame.GasLimits.Execution {
			return fmt.Errorf("%w: total frame gas too high", ErrFrameTxInvalidFormat)
		}
		totalFrameGas += frame.GasLimits.Execution
		if gomath.MaxUint64-totalFrameGas < frame.GasLimits.State {
			return fmt.Errorf("%w: total frame gas too high", ErrFrameTxInvalidFormat)
		}
		totalFrameGas += frame.GasLimits.State

		// Execution approval is only allowed for frames that resolve to
		// the transaction sender.
		if frame.Flags&ApproveExecution != 0 && frame.ResolvedTarget(tx.Sender) != tx.Sender {
			return fmt.Errorf("%w: execution approval flag outside sender target", ErrFrameTxInvalidFormat)
		}

		// An atomic batch must be terminated by a subsequent non-VERIFY
		// frame and can never contain a VERIFY frame.
		if frame.Flags&AtomicBatchFlag != 0 {
			if frame.Mode == ModeVerify {
				return fmt.Errorf("%w: atomic batches cannot contain verify frames", ErrFrameTxInvalidFormat)
			}
			if i+1 >= len(tx.Frames) {
				return fmt.Errorf("%w: atomic batch flag set on last frame", ErrFrameTxInvalidFormat)
			}
			if tx.Frames[i+1].Mode == ModeVerify {
				return fmt.Errorf("%w: atomic batches cannot contain verify frames", ErrFrameTxInvalidFormat)
			}
		}

		// Approval scope is disallowed on every frame of an atomic batch,
		// including its terminating frame. A frame belongs to a batch when
		// it or its predecessor carries the flag.
		inBatch := frame.Flags&AtomicBatchFlag != 0 ||
			(i > 0 && tx.Frames[i-1].Flags&AtomicBatchFlag != 0)
		if inBatch && frame.Flags&ApproveScopeMask != 0 {
			return fmt.Errorf("%w: atomic batch frames cannot carry approval scope", ErrFrameTxInvalidFormat)
		}

		if frame.IsExpiryVerifier() {
			expiryVerifierFrames++
			if frame.Flags != 0 {
				return fmt.Errorf("%w: expiry verifier frame flags must be zero", ErrFrameTxInvalidFormat)
			}
			if !frame.Value.IsZero() {
				return fmt.Errorf("%w: expiry verifier frame with value", ErrFrameTxInvalidFormat)
			}
			if frame.GasLimits.State != 0 {
				return fmt.Errorf("%w: expiry verifier frame with state gas", ErrFrameTxInvalidFormat)
			}
			if len(frame.Data) != params.FrameTxExpiryDataLen {
				return fmt.Errorf("%w: expiry verifier frame data must be 8 bytes", ErrFrameTxInvalidFormat)
			}
		}
	}
	if expiryVerifierFrames > 1 {
		return fmt.Errorf("%w: multiple expiry verifier frames", ErrFrameTxInvalidFormat)
	}
	if len(tx.BlobVersionedHashes) == 0 && !tx.Fees.MaxFeePerBlobGas.IsZero() {
		return fmt.Errorf("%w: max fee per blob gas must be zero without blobs", ErrFrameTxInvalidFormat)
	}
	if tx.Nonce == gomath.MaxUint64 {
		return fmt.Errorf("%w: nonce overflow", ErrFrameTxInvalidFormat)
	}
	// The per-transaction gas cap of EIP-7825 bounds the execution
	// dimension alone: the intrinsic cost plus the frames' execution
	// budgets, with the calldata floor checked against the same cap. State
	// gas is bounded only by the encoding limit and the block's state gas
	// capacity.
	intrinsicGas, err := FrameTxIntrinsicGas(tx.Frames, tx.Signatures, tx.Sender)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFrameTxInvalidFormat, err)
	}
	floorGas, err := FrameTxFloorGas(tx.Frames, tx.Signatures, tx.Sender)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFrameTxInvalidFormat, err)
	}
	executionGas, _ := FrameTxBudgetTotals(tx.Frames)
	if gomath.MaxUint64-intrinsicGas < executionGas {
		return fmt.Errorf("%w: total frame gas too high", ErrFrameTxInvalidFormat)
	}
	if max(intrinsicGas+executionGas, floorGas) > params.MaxTxGas {
		return fmt.Errorf("%w: derived execution gas limit exceeds the transaction gas cap", ErrFrameTxInvalidFormat)
	}
	return nil
}

// sigHash returns the canonical signature hash of the frame transaction.
// The raw signature bytes of every entry with an empty msg are elided.
func (tx *FrameTx) sigHash(chainID *big.Int) common.Hash {
	if chainID == nil {
		chainID = tx.chainID()
	}
	sigs := make(SignatureList, len(tx.Signatures))
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
			tx.Fees,
			tx.BlobVersionedHashes,
		},
	)
}

func (tx *FrameTx) copy() TxData {
	cpy := &FrameTx{
		Nonce:               tx.Nonce,
		Sender:              tx.Sender,
		Frames:              make([]Frame, len(tx.Frames)),
		Signatures:          make(SignatureList, len(tx.Signatures)),
		BlobVersionedHashes: make([]common.Hash, len(tx.BlobVersionedHashes)),

		ChainID: new(uint256.Int),
		Fees: Fees{
			MaxPriorityFeePerGas: new(uint256.Int),
			MaxFeePerGas:         new(uint256.Int),
			MaxFeePerBlobGas:     new(uint256.Int),
		},
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
		cpy.Frames[i] = Frame{
			Mode:      frame.Mode,
			Flags:     frame.Flags,
			Target:    target,
			GasLimits: frame.GasLimits,
			Value:     value,
			Data:      common.CopyBytes(frame.Data),
		}
	}
	for i, sig := range tx.Signatures {
		cpy.Signatures[i] = SignatureEntry{
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
	if tx.Fees.MaxPriorityFeePerGas != nil {
		cpy.Fees.MaxPriorityFeePerGas.Set(tx.Fees.MaxPriorityFeePerGas)
	}
	if tx.Fees.MaxFeePerGas != nil {
		cpy.Fees.MaxFeePerGas.Set(tx.Fees.MaxFeePerGas)
	}
	if tx.Fees.MaxFeePerBlobGas != nil {
		cpy.Fees.MaxFeePerBlobGas.Set(tx.Fees.MaxFeePerBlobGas)
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
	if tx.Fees.MaxFeePerGas == nil {
		return new(big.Int)
	}
	return tx.Fees.MaxFeePerGas.ToBig()
}
func (tx *FrameTx) gasTipCap() *big.Int {
	if tx.Fees.MaxPriorityFeePerGas == nil {
		return new(big.Int)
	}
	return tx.Fees.MaxPriorityFeePerGas.ToBig()
}
func (tx *FrameTx) gasPrice() *big.Int {
	if tx.Fees.MaxFeePerGas == nil {
		return new(big.Int)
	}
	return tx.Fees.MaxFeePerGas.ToBig()
}

// gas returns the inclusion-facing derived gas limit of the frame
// transaction: its max_gas anchor.
func (tx *FrameTx) gas() uint64 {
	total, err := FrameTxMaxGas(tx.Frames, tx.Signatures, tx.Sender)
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
	if tx.ChainID == nil || tx.Fees.MaxPriorityFeePerGas == nil || tx.Fees.MaxFeePerGas == nil || tx.Fees.MaxFeePerBlobGas == nil {
		return ErrFrameTxInvalidFormat
	}
	return nil
}

// Frame execution modes (EIP-8141).
const (
	ModeDefault uint64 = 0
	ModeVerify  uint64 = 1
	ModeSender  uint64 = 2
)

// Frame flags (EIP-8141). Bits 0-1 hold the allowed approval scope, bit 2
// marks the frame as part of an atomic batch. Higher bits are reserved.
const (
	ApproveNone                uint64 = 0x0
	ApprovePayment             uint64 = 0x1
	ApproveExecution           uint64 = 0x2
	ApproveExecutionAndPayment uint64 = 0x3
	ApproveScopeMask           uint64 = 0x3
	AtomicBatchFlag            uint64 = 0x4
	FlagsLimit                 uint64 = 0x8
)

// Frame is a single frame in a frame transaction.
type Frame struct {
	Mode      uint64
	Flags     uint64
	Target    *common.Address `rlp:"nil"` // nil resolves to the transaction sender
	GasLimits Limits
	Value     *uint256.Int
	Data      []byte
}

// ResolvedTarget returns the frame's target, resolving a nil target to the
// transaction sender.
func (f *Frame) ResolvedTarget(sender common.Address) common.Address {
	if f.Target == nil {
		return sender
	}
	return *f.Target
}

// IsExpiryVerifier reports whether the frame is an expiry verifier frame,
// i.e. a VERIFY frame targeting the expiry verifier contract.
func (f *Frame) IsExpiryVerifier() bool {
	return f.Mode == ModeVerify && f.Target != nil && *f.Target == params.FrameTxExpiryVerifier
}

// The JSON codec accepts hex numbers with leading zero digits, as emitted by
// test fixtures, and marshals them canonically.
type frameJSON struct {
	Mode          math.HexOrDecimal64   `json:"mode"`
	Flags         math.HexOrDecimal64   `json:"flags"`
	Target        *common.Address       `json:"target,omitempty"`
	GasLimit      math.HexOrDecimal64   `json:"gasLimit"`
	StateGasLimit math.HexOrDecimal64   `json:"stateGasLimit"`
	Value         *math.HexOrDecimal256 `json:"value"`
	Data          hexutil.Bytes         `json:"data"`
}

func (f Frame) MarshalJSON() ([]byte, error) {
	enc := frameJSON{
		Mode:          math.HexOrDecimal64(f.Mode),
		Flags:         math.HexOrDecimal64(f.Flags),
		Target:        f.Target,
		GasLimit:      math.HexOrDecimal64(f.GasLimits.Execution),
		StateGasLimit: math.HexOrDecimal64(f.GasLimits.State),
		Data:          f.Data,
	}
	if f.Value != nil {
		enc.Value = (*math.HexOrDecimal256)(f.Value.ToBig())
	}
	return json.Marshal(&enc)
}

func (f *Frame) UnmarshalJSON(input []byte) error {
	var dec frameJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	f.Mode = uint64(dec.Mode)
	f.Flags = uint64(dec.Flags)
	f.Target = dec.Target
	f.GasLimits = Limits{
		Execution: uint64(dec.GasLimit),
		State:     uint64(dec.StateGasLimit),
	}
	f.Value = new(uint256.Int)
	if dec.Value != nil {
		value, overflow := uint256.FromBig((*big.Int)(dec.Value))
		if overflow {
			return errors.New("frame value exceeds 256 bits")
		}
		f.Value = value
	}
	f.Data = dec.Data
	return nil
}

// Limits is the gas budget pair of a frame, one per gas dimension.
// The two budgets are independent: neither dimension can fund charges of the
// other, and unused gas in one is not available to the other. It encodes as
// the nested `limits = [execution, state]` list of the frame payload.
type Limits struct {
	Execution uint64
	State     uint64
}

// SignatureList is the signature list of a frame transaction.
type SignatureList []SignatureEntry

// SignatureEntry is the element type of a signature list.
type SignatureEntry struct {
	Scheme    uint64
	Signer    []byte // 20-byte address for SECP256K1 and P256, empty for ARBITRARY
	Msg       []byte // empty (canonical signature hash) or an explicit 32-byte digest
	Signature []byte
}

type frameTxSignatureJSON struct {
	Scheme    math.HexOrDecimal64 `json:"scheme"`
	Signer    hexutil.Bytes       `json:"signer"`
	Msg       hexutil.Bytes       `json:"msg"`
	Signature hexutil.Bytes       `json:"signature"`
}

func (s SignatureEntry) MarshalJSON() ([]byte, error) {
	enc := frameTxSignatureJSON{
		Scheme:    math.HexOrDecimal64(s.Scheme),
		Signer:    s.Signer,
		Msg:       s.Msg,
		Signature: s.Signature,
	}
	return json.Marshal(&enc)
}

func (s *SignatureEntry) UnmarshalJSON(input []byte) error {
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

// ResolvedSigner returns the signer address of a protocol-validated
// signature entry, resolving an empty signer to the transaction sender.
func (s *SignatureEntry) ResolvedSigner(sender common.Address) common.Address {
	if len(s.Signer) == 0 {
		return sender
	}
	return common.BytesToAddress(s.Signer)
}

// Fees groups the fee parameters of a frame transaction, encoded as
// the nested `fees` list of the transaction payload.
type Fees struct {
	MaxPriorityFeePerGas *uint256.Int
	MaxFeePerGas         *uint256.Int
	MaxFeePerBlobGas     *uint256.Int
}

// FrameTxSignatureGas returns the gas charged for the protocol validation of
// a single signature entry.
func FrameTxSignatureGas(sig *SignatureEntry) uint64 {
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
func FrameTxChargedData(frames []Frame, sigs SignatureList) [][]byte {
	charged := make([][]byte, 0, len(frames)+3*len(sigs))
	for i := range frames {
		charged = append(charged, frames[i].Data)
	}
	for i := range sigs {
		charged = append(charged, sigs[i].Signer, sigs[i].Msg, sigs[i].Signature)
	}
	return charged
}

// FrameTxBudgetTotals returns the frames' total declared gas budget in each
// dimension. Overflow is guarded by ValidateStatic, which bounds the sum of
// both dimensions.
func FrameTxBudgetTotals(frames []Frame) (execution, state uint64) {
	for i := range frames {
		execution += frames[i].GasLimits.Execution
		state += frames[i].GasLimits.State
	}
	return execution, state
}

// frameTxMandatoryGas computes the costs a frame transaction always pays
// regardless of execution: the base cost, the per-frame cost, the signature
// verification cost, and the value transfer cost of each value-bearing frame
// with an explicit target other than the sender.
func frameTxMandatoryGas(frames []Frame, sigs SignatureList, sender common.Address) uint64 {
	gas := params.FrameTxIntrinsicGas + uint64(len(frames))*params.FrameTxPerFrameGas
	for i := range sigs {
		gas += FrameTxSignatureGas(&sigs[i])
	}
	for i := range frames {
		frame := &frames[i]
		if frame.Value != nil && !frame.Value.IsZero() && frame.Target != nil && *frame.Target != sender {
			gas += params.TxValueCost2780
		}
	}
	return gas
}

// FrameTxIntrinsicGas computes the intrinsic execution gas of an EIP-8141
// frame transaction: the mandatory costs plus the standard calldata cost of
// the frame and signature byte fields. Unlike other transaction types there
// is no recipient component: target access is paid during frame execution
// from each frame's own execution gas budget.
func FrameTxIntrinsicGas(frames []Frame, sigs SignatureList, sender common.Address) (uint64, error) {
	gas := frameTxMandatoryGas(frames, sigs, sender)
	var dataLen, z uint64
	for _, d := range FrameTxChargedData(frames, sigs) {
		dataLen += uint64(len(d))
		z += uint64(bytes.Count(d, []byte{0}))
	}
	if dataLen > 0 {
		nz := dataLen - z
		// Frame transactions exist only post-Istanbul.
		nonZeroGas := params.TxDataNonZeroGasEIP2028
		if (gomath.MaxUint64-gas)/nonZeroGas < nz {
			return 0, errors.New("gas uint64 overflow")
		}
		gas += nz * nonZeroGas
		if (gomath.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, errors.New("gas uint64 overflow")
		}
		gas += z * params.TxDataZeroGas
	}
	return gas, nil
}

// FrameTxFloorGas computes the calldata floor of a frame transaction per
// EIP-7623 and EIP-7976: every charged byte counts as a standard token
// priced at the floor token cost, uniformly and independently of its value.
// The floor is anchored on the mandatory costs, so it never undercuts the
// transaction's own intrinsic base.
func FrameTxFloorGas(frames []Frame, sigs SignatureList, sender common.Address) (uint64, error) {
	var dataLen uint64
	for _, data := range FrameTxChargedData(frames, sigs) {
		dataLen += uint64(len(data))
	}
	if gomath.MaxUint64/(params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken7976) < dataLen {
		return 0, errors.New("gas uint64 overflow")
	}
	floorGas := frameTxMandatoryGas(frames, sigs, sender)
	dataGas := dataLen * params.TxTokenPerNonZeroByte * params.TxCostFloorPerToken7976
	if gomath.MaxUint64-floorGas < dataGas {
		return 0, errors.New("gas uint64 overflow")
	}
	return floorGas + dataGas, nil
}

// FrameTxStandardGasLimit computes the settlement anchor of a frame
// transaction: its intrinsic execution gas plus the sum of the frames' gas
// budgets in both dimensions.
func FrameTxStandardGasLimit(frames []Frame, sigs SignatureList, sender common.Address) (uint64, error) {
	intrinsicGas, err := FrameTxIntrinsicGas(frames, sigs, sender)
	if err != nil {
		return 0, err
	}
	executionGas, stateGas := FrameTxBudgetTotals(frames)
	total := intrinsicGas
	for _, budget := range []uint64{executionGas, stateGas} {
		if gomath.MaxUint64-total < budget {
			return 0, errors.New("gas uint64 overflow")
		}
		total += budget
	}
	return total, nil
}

// FrameTxMaxGas computes the inclusion anchor of a frame transaction: the
// larger of its standard gas limit and its calldata floor plus the frames'
// total state gas budget. The maximum transaction cost escrowed from the
// payer prices this anchor at the fee cap.
func FrameTxMaxGas(frames []Frame, sigs SignatureList, sender common.Address) (uint64, error) {
	standard, err := FrameTxStandardGasLimit(frames, sigs, sender)
	if err != nil {
		return 0, err
	}
	floorGas, err := FrameTxFloorGas(frames, sigs, sender)
	if err != nil {
		return 0, err
	}
	_, stateGas := FrameTxBudgetTotals(frames)
	if gomath.MaxUint64-floorGas < stateGas {
		return 0, errors.New("gas uint64 overflow")
	}
	return max(standard, floorGas+stateGas), nil
}

// ValidateFrameTxSignatures validates all signature entries of a frame
// transaction against the canonical signature hash, per EIP-8141.
func ValidateFrameTxSignatures(sigs SignatureList, sender common.Address, sigHash common.Hash) error {
	for i := range sigs {
		if !validateFrameTxSignature(&sigs[i], sender, sigHash) {
			return fmt.Errorf("%w: entry %d", ErrFrameTxInvalidSignature, i)
		}
	}
	return nil
}

func validateFrameTxSignature(sig *SignatureEntry, sender common.Address, sigHash common.Hash) bool {
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
		// r and s must be canonical, with low-s, so each signature has one
		// encoding. P256 verification itself accepts high-s values.
		if r.Sign() <= 0 || s.Sign() <= 0 || r.Cmp(secp256r1N) >= 0 || s.Cmp(secp256r1HalfN) > 0 {
			return false
		}
		return secp256r1.Verify(msg[:], r, s, x, y)

	case FrameTxSchemeArbitrary:
		return len(sig.Signer) == 0

	default:
		return false
	}
}
