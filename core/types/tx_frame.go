// Copyright 2025 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// Frame mode constants as defined by EIP-8141.
const (
	FrameModeDefault uint8 = 0 // Regular call with ENTRY_POINT as caller.
	FrameModeVerify  uint8 = 1 // Validation frame; must call APPROVE; STATICCALL. Data elided from sig hash.
	FrameModeSender  uint8 = 2 // Regular call with tx.sender as caller; requires sender_approved.
)

// Frame represents a single call frame within a Frame Transaction
type Frame struct {
	Mode     uint8
	Target   common.Address
	GasLimit uint64
	Data     []byte
}

// FrameTx represents an EIP-8141 Frame Transaction.
type FrameTx struct {
	ChainID   *uint256.Int
	Nonce     uint64
	Sender    common.Address
	Frames    []Frame
	GasTipCap *uint256.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap *uint256.Int // a.k.a. maxFeePerGas

	// Optional blob fields
	BlobFeeCap *uint256.Int  // a.k.a. maxFeePerBlobGas
	BlobHashes []common.Hash // Blob versioned hashes
}

// Todo(jvn): Just a quick version of each fn to fullfill requirements
// Must chk correctness later once impl completes

// copy creates a deep copy of the transaction data.
func (tx *FrameTx) copy() TxData {
	cpy := &FrameTx{
		Nonce:     tx.Nonce,
		Sender:    tx.Sender,
		Frames:    make([]Frame, len(tx.Frames)),
		ChainID:   new(uint256.Int),
		GasTipCap: new(uint256.Int),
		GasFeeCap: new(uint256.Int),
	}
	// Deep copy frames.
	for i, f := range tx.Frames {
		cpy.Frames[i] = Frame{
			Mode:     f.Mode,
			Target:   f.Target,
			GasLimit: f.GasLimit,
			Data:     common.CopyBytes(f.Data),
		}
	}
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
		cpy.BlobFeeCap = new(uint256.Int)
		cpy.BlobFeeCap.Set(tx.BlobFeeCap)
	}
	if len(tx.BlobHashes) > 0 {
		cpy.BlobHashes = make([]common.Hash, len(tx.BlobHashes))
		copy(cpy.BlobHashes, tx.BlobHashes)
	}
	return cpy
}

// accessors for TxData interface.
func (tx *FrameTx) txType() byte           { return FrameTxType }
func (tx *FrameTx) chainID() *big.Int      { return tx.ChainID.ToBig() }
func (tx *FrameTx) accessList() AccessList { return nil }
func (tx *FrameTx) data() []byte           { return nil }

// todo(jvn) change this
func (tx *FrameTx) gasFeeCap() *big.Int { return tx.GasFeeCap.ToBig() }
func (tx *FrameTx) gasTipCap() *big.Int { return tx.GasTipCap.ToBig() }
func (tx *FrameTx) gasPrice() *big.Int  { return tx.GasFeeCap.ToBig() }
func (tx *FrameTx) value() *big.Int     { return new(big.Int) } // As Spec Frame txs have no value field Ig , confirm later
func (tx *FrameTx) nonce() uint64       { return tx.Nonce }
func (tx *FrameTx) to() *common.Address { return nil } // No single target; each frame has its own.

// Todo: StreamLine Gas Calc , currently spreadout here and in State Processor
// Port correct Implementation here to keep it clean
func (tx *FrameTx) gas() uint64 {
	total := uint64(params.FrameTxIntrinsicGas)
	total += tx.CalldataCost()
	for _, f := range tx.Frames {
		total += f.GasLimit
	}
	return total
}

func (tx *FrameTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap.ToBig())
	}
	tip := dst.Sub(tx.GasFeeCap.ToBig(), baseFee)
	if tip.Cmp(tx.GasTipCap.ToBig()) > 0 {
		tip.Set(tx.GasTipCap.ToBig())
	}
	return tip.Add(tip, baseFee)
}

func (tx *FrameTx) rawSignatureValues() (v, r, s *big.Int) {
	// Frame transactions have no ECDSA signature.
	return new(big.Int), new(big.Int), new(big.Int)
}

func (tx *FrameTx) setSignatureValues(chainID, v, r, s *big.Int) {
	// Frame transactions don't have signatures.
	tx.ChainID = uint256.MustFromBig(chainID)
}

func (tx *FrameTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *FrameTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}

// sigHash computes the signature hash for a frame transaction.
func (tx *FrameTx) sigHash(chainID *big.Int) common.Hash {
	elidedFrames := make([]interface{}, len(tx.Frames))
	for i, f := range tx.Frames {
		frameData := f.Data
		if f.Mode == FrameModeVerify {
			frameData = []byte{} // Elide data for VERIFY frames.
		}
		elidedFrames[i] = []interface{}{
			f.Mode,
			f.Target,
			f.GasLimit,
			frameData,
		}
	}
	return prefixedRlpHash(
		FrameTxType,
		[]interface{}{
			chainID,
			tx.Nonce,
			tx.Sender,
			elidedFrames,
			tx.GasTipCap,
			tx.GasFeeCap,
			tx.BlobFeeCap,
			tx.BlobHashes,
		})
}

// ComputeSigHash returns the signature hash for external use.
func (tx *FrameTx) ComputeSigHash() common.Hash {
	return tx.sigHash(tx.ChainID.ToBig())
}

// TotalFrameGas returns the sum of gas limits across all frames.
func (tx *FrameTx) TotalFrameGas() uint64 {
	var total uint64
	for _, f := range tx.Frames {
		total += f.GasLimit
	}
	return total
}

// CalldataCost returns the gas cost for the calldata equivalent in the frame transaction.
func (tx *FrameTx) CalldataCost() uint64 {
	encodedFrames, err := rlp.EncodeToBytes(tx.Frames)
	if err != nil {
		return 0
	}
	return calldataCost(encodedFrames)
}

// Check this dont think Currently using this anywhere
func calldataCost(data []byte) uint64 {
	z := uint64(bytes.Count(data, []byte{0}))
	nz := uint64(len(data)) - z
	return z*params.TxDataZeroGas + nz*params.TxDataNonZeroGasEIP2028
}

// FrameSigHash returns the signature hash
func (tx *Transaction) FrameSigHash() common.Hash {
	if ftx, ok := tx.inner.(*FrameTx); ok {
		return ftx.ComputeSigHash()
	}
	return common.Hash{}
}

// Frames returns the frame list for frame transactions, nil otherwise.
func (tx *Transaction) Frames() []Frame {
	if ftx, ok := tx.inner.(*FrameTx); ok {
		return ftx.Frames
	}
	return nil
}

// FrameSender returns the explicit sender for frame transactions.
func (tx *Transaction) FrameSender() (common.Address, bool) {
	if ftx, ok := tx.inner.(*FrameTx); ok {
		return ftx.Sender, true
	}
	return common.Address{}, false
}
