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

package history

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// HistoryMode configures history pruning.
type HistoryMode uint32

const (
	// KeepAll (default) means that all chain history down to genesis block will be kept.
	KeepAll HistoryMode = iota

	// KeepPostMerge sets the history pruning point to the merge activation block.
	KeepPostMerge

	// KeepPostCancun sets the history pruning point to the Cancun (Dencun) activation block.
	KeepPostCancun
)

func (m HistoryMode) IsValid() bool {
	return m <= KeepPostCancun
}

func (m HistoryMode) String() string {
	switch m {
	case KeepAll:
		return "all"
	case KeepPostMerge:
		return "postmerge"
	case KeepPostCancun:
		return "postcancun"
	default:
		return fmt.Sprintf("invalid HistoryMode(%d)", m)
	}
}

// MarshalText implements encoding.TextMarshaler.
func (m HistoryMode) MarshalText() ([]byte, error) {
	if m.IsValid() {
		return []byte(m.String()), nil
	}
	return nil, fmt.Errorf("unknown history mode %d", m)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *HistoryMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "all":
		*m = KeepAll
	case "postmerge":
		*m = KeepPostMerge
	case "postcancun":
		*m = KeepPostCancun
	default:
		return fmt.Errorf(`unknown history mode %q, want "all", "postmerge", or "postcancun"`, text)
	}
	return nil
}

type PrunePoint struct {
	BlockNumber uint64
	BlockHash   common.Hash
}

// MergePrunePoints contains the pre-defined history pruning cutoff blocks for known networks.
// They point to the first post-merge block. Any pruning should truncate *up to* but excluding
// the given block.
var MergePrunePoints = map[common.Hash]*PrunePoint{
	// mainnet
	params.MainnetGenesisHash: {
		BlockNumber: 15537393,
		BlockHash:   common.HexToHash("0x55b11b918355b1ef9c5db810302ebad0bf2544255b530cdce90674d5887bb286"),
	},
	// sepolia
	params.SepoliaGenesisHash: {
		BlockNumber: 1450409,
		BlockHash:   common.HexToHash("0x229f6b18ca1552f1d5146deceb5387333f40dc6275aebee3f2c5c4ece07d02db"),
	},
}

// CancunPrunePoints contains the pre-defined history pruning cutoff blocks for the Cancun
// (Dencun) upgrade. They point to the first post-Cancun block. Any pruning should truncate
// *up to* but excluding the given block.
var CancunPrunePoints = map[common.Hash]*PrunePoint{
	// mainnet - first Cancun block (March 13, 2024)
	params.MainnetGenesisHash: {
		BlockNumber: 19426587,
		BlockHash:   common.HexToHash("0xf8e2f40d98fe5862bc947c8c83d34799c50fb344d7445d020a8a946d891b62ee"),
	},
	// sepolia - first Cancun block (January 30, 2024)
	params.SepoliaGenesisHash: {
		BlockNumber: 5187023,
		BlockHash:   common.HexToHash("0x8f9753667f95418f70db36279a269ed6523cea399ecc3f4cfa2f1689a3a4b130"),
	},
}

// PrunePoints is an alias for MergePrunePoints for backward compatibility.
// Deprecated: Use GetPrunePoint or MergePrunePoints directly.
var PrunePoints = MergePrunePoints

// GetPrunePoint returns the prune point for the given genesis hash and history mode.
// Returns nil if no prune point is defined for the given combination.
func GetPrunePoint(genesisHash common.Hash, mode HistoryMode) *PrunePoint {
	switch mode {
	case KeepPostMerge:
		return MergePrunePoints[genesisHash]
	case KeepPostCancun:
		return CancunPrunePoints[genesisHash]
	default:
		return nil
	}
}

// PrunedHistoryError is returned by APIs when the requested history is pruned.
type PrunedHistoryError struct{}

func (e *PrunedHistoryError) Error() string  { return "pruned history unavailable" }
func (e *PrunedHistoryError) ErrorCode() int { return 4444 }
