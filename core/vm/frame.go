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

package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EIP-8141 scope constants.
const (
	ApproveNone                = 0x0
	ApprovePayment             = 0x1
	ApproveExecution           = 0x2
	ApproveExecutionAndPayment = 0x3
	ApproveScopeMask           = 0x3
)

// FrameTxState is the transaction-scoped approval context shared across all
// frames of a frame transaction.
type FrameTxState struct {
	SenderApproved bool
	Payer          common.Address
	PayerSet       bool
	MaxCost        *big.Int
}

// FrameContext carries the EIP-8141 frame transaction state to the EVM so that
// the frame introspection and APPROVE instructions can access it. It is set up
// by the state transition before each frame is executed.
type FrameContext struct {
	Tx      *types.FrameTx
	SigHash common.Hash
	MaxGas  uint64
	State   *FrameTxState // shared across all frames of the transaction

	// Per-frame state, updated before each frame's top-level call.
	FrameIndex     int
	Frame          *types.Frame
	ResolvedTarget common.Address

	// FrameStatuses records the execution status of each frame (0 = failed,
	// 1 = success, 2 = skipped due to a failed atomic batch).
	FrameStatuses []byte
}

// SetFrameContext installs the frame transaction context on the EVM. When nil
// is passed, any previously installed context is cleared and the frame
// instructions behave as if executing outside a frame transaction.
func (evm *EVM) SetFrameContext(fc *FrameContext) {
	evm.frameCtx = fc
}

// Frame returns the active frame context, or nil when the EVM is not executing
// a frame transaction.
func (evm *EVM) Frame() *FrameContext {
	return evm.frameCtx
}

// resolvedTargetFor returns the resolved target address of the given frame.
func (fc *FrameContext) resolvedTargetFor(index int) common.Address {
	frame := &fc.Tx.Frames[index]
	if frame.Target == nil {
		return fc.Tx.Sender
	}
	return *frame.Target
}

// frameStatus returns the recorded status of an already-executed frame: 0 for
// failure, 1 for success, 2 for skipped due to a failed atomic batch.
func (fc *FrameContext) frameStatus(index int) byte {
	return fc.FrameStatuses[index]
}
