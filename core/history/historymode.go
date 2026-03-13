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

	// KeepPostPrague sets the history pruning point to the Prague (Pectra) activation block.
	KeepPostPrague
)

func (m HistoryMode) IsValid() bool {
	return m <= KeepPostPrague
}

func (m HistoryMode) String() string {
	switch m {
	case KeepAll:
		return "all"
	case KeepPostMerge:
		return "postmerge"
	case KeepPostPrague:
		return "postprague"
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
	case "postprague":
		*m = KeepPostPrague
	default:
		return fmt.Errorf(`unknown history mode %q, want "all", "postmerge", or "postprague"`, text)
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

// PraguePrunePoints contains the pre-defined history pruning cutoff blocks for the Prague
// (Pectra) upgrade. They point to the first post-Prague block. Any pruning should truncate
// *up to* but excluding the given block.
var PraguePrunePoints = map[common.Hash]*PrunePoint{
	// mainnet - first Prague block (May 7, 2025)
	params.MainnetGenesisHash: {
		BlockNumber: 22431084,
		BlockHash:   common.HexToHash("0x50c8cab760b2948349c590461b166773c45d8f4858cccf5a43025ab2960152e8"),
	},
	// sepolia - first Prague block (March 5, 2025)
	params.SepoliaGenesisHash: {
		BlockNumber: 7836331,
		BlockHash:   common.HexToHash("0xe6571beb68bf24dbd8a6ba354518996920c55a3f8d8fdca423e391b8ad071f22"),
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
	case KeepPostPrague:
		return PraguePrunePoints[genesisHash]
	default:
		return nil
	}
}

// PrunedHistoryError is returned by APIs when the requested history is pruned.
type PrunedHistoryError struct{}

func (e *PrunedHistoryError) Error() string  { return "pruned history unavailable" }
func (e *PrunedHistoryError) ErrorCode() int { return 4444 }
