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

// Package framepool implements public-mempool policy for frame transactions.
package framepool

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Public-mempool policy limits from EIP-8141 (not consensus parameters).
const (
	MaxVerifyGas                            uint64 = 100_000
	MaxVerifyStateGas                       uint64 = 500_000
	MaxPendingTxsUsingNonCanonicalPaymaster        = 1
)

var (
	ErrInvalidPrefix  = errors.New("unrecognized frame validation prefix")
	ErrPrefixGasLimit = errors.New("frame validation prefix exceeds gas limit")
)

// Prefix describes a recognized validation prefix. Optional frame indexes are
// -1 when absent. End is the inclusive index of the payment-approving frame,
// always within the classified frame list, and Payer is that frame's resolved
// target, which APPROVE makes the payer. ExecutionGas includes all
// signature-validation gas, while StateGas includes only the declared prefix
// state budgets. ExpiryDeadline is meaningful only when ExpiryFrame is present;
// equality with the head timestamp is valid. Recent roots expire at
// OldestRootSlot + recentRootLength; NewestRootSlot also bounds rollback validity.
type Prefix struct {
	ExpiryFrame     int
	ExpiryDeadline  uint64
	RecentRootFrame int
	OldestRootSlot  uint64
	NewestRootSlot  uint64
	DeployFrame     int
	VerifyFrame     int
	PayFrame        int
	End             int
	Payer           common.Address
	ExecutionGas    uint64
	StateGas        uint64
}

// ClassifyPrefix recognizes the public-mempool validation shapes without state
// access or signature verification. Consensus static validation is separate.
// A deploy follows the optional expiry and recent-root verifiers. Later DEFAULT
// frames are post-ops, not deploys.
func ClassifyPrefix(frames []types.Frame, sender common.Address, signatures types.SignatureList) (Prefix, error) {
	prefix := Prefix{ExpiryFrame: -1, RecentRootFrame: -1, DeployFrame: -1, VerifyFrame: -1, PayFrame: -1, End: -1}
	invalid := func(reason string) (Prefix, error) {
		return Prefix{}, fmt.Errorf("%w: %s", ErrInvalidPrefix, reason)
	}
	start := 0
	for i := range frames {
		if frames[i].IsExpiryVerifier() {
			if i != 0 || frames[i].Flags != 0 || len(frames[i].Data) != params.FrameTxExpiryDataLen {
				return invalid("expiry must be first with zero flags and an 8-byte deadline")
			}
			prefix.ExpiryFrame = i
			prefix.ExpiryDeadline = binary.BigEndian.Uint64(frames[i].Data)
			start = 1
		}
	}
	for i := range frames {
		frame := &frames[i]
		if frame.Mode != types.ModeVerify || frame.ResolvedTarget(sender) != params.RecentRootAddress {
			continue
		}
		if prefix.RecentRootFrame >= 0 || i != start || frame.Flags != 0 || (frame.Value != nil && !frame.Value.IsZero()) || frame.GasLimits.State != 0 ||
			len(frame.Data) == 0 || len(frame.Data)%recentRootTupleBytes != 0 || len(frame.Data) > maxRecentRootReferences*recentRootTupleBytes {
			return invalid("invalid recent root verifier shape or position")
		}
		prefix.RecentRootFrame = i
		prefix.OldestRootSlot = ^uint64(0)
		for offset := 0; offset < len(frame.Data); offset += recentRootTupleBytes {
			slot := binary.BigEndian.Uint64(frame.Data[offset+32 : offset+40])
			prefix.OldestRootSlot = min(prefix.OldestRootSlot, slot)
			prefix.NewestRootSlot = max(prefix.NewestRootSlot, slot)
		}
		start++
	}
	if start < len(frames) && frames[start].Mode == types.ModeDefault && frames[start].Flags == 0 {
		prefix.DeployFrame = start
		start++
	}
	if start >= len(frames) {
		return invalid("missing sender verification")
	}
	verify := &frames[start]
	if verify.Mode != types.ModeVerify || verify.ResolvedTarget(sender) != sender {
		return invalid("sender verification must target sender")
	}
	prefix.VerifyFrame = start
	switch verify.Flags {
	case types.ApproveExecutionAndPayment:
		prefix.End = start
	case types.ApproveExecution:
		pay := start + 1
		if pay >= len(frames) || frames[pay].Mode != types.ModeVerify || frames[pay].Flags != types.ApprovePayment {
			return invalid("missing payment verification")
		}
		prefix.PayFrame, prefix.End = pay, pay
	default:
		return invalid("invalid sender verification flags")
	}
	prefix.Payer = frames[prefix.End].ResolvedTarget(sender)
	postTx := false
	for i := range frames {
		if frames[i].Mode == types.ModePostTx {
			postTx = true
		} else if postTx {
			return invalid("execution frame after POST_TX")
		}
		if i <= prefix.End {
			if frames[i].Flags&types.AtomicBatchFlag != 0 {
				return invalid("atomic batch in validation prefix")
			}
			// Subtract before adding so even malicious uint64 budgets cannot
			// wrap the totals. The running totals never exceed their caps.
			if frames[i].GasLimits.Execution > MaxVerifyGas-prefix.ExecutionGas || frames[i].GasLimits.State > MaxVerifyStateGas-prefix.StateGas {
				return Prefix{}, ErrPrefixGasLimit
			}
			prefix.ExecutionGas += frames[i].GasLimits.Execution
			prefix.StateGas += frames[i].GasLimits.State
		} else if frames[i].Mode == types.ModeVerify {
			return invalid("VERIFY after validation prefix")
		}
	}
	for i := range signatures {
		gas := types.FrameTxSignatureGas(&signatures[i])
		if gas > MaxVerifyGas-prefix.ExecutionGas {
			return Prefix{}, ErrPrefixGasLimit
		}
		prefix.ExecutionGas += gas
	}
	return prefix, nil
}
