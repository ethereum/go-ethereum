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

package ethconfig

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// HistoryMode configures history pruning.
type HistoryMode uint32

const (
	// AllHistory (default) means that all chain history down to genesis block will be kept.
	AllHistory HistoryMode = iota

	// PostMergeHistory sets the history pruning point to the merge activation block.
	PostMergeHistory
)

func (m HistoryMode) IsValid() bool {
	return m <= PostMergeHistory
}

func (m HistoryMode) String() string {
	switch m {
	case AllHistory:
		return "all"
	case PostMergeHistory:
		return "postmerge"
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
		*m = AllHistory
	case "postmerge":
		*m = PostMergeHistory
	default:
		return fmt.Errorf(`unknown sync mode %q, want "all" or "postmerge"`, text)
	}
	return nil
}

type HistoryPrunePoint struct {
	BlockNumber uint64
	BlockHash   common.Hash
}

// HistoryPrunePoints contains the pre-defined history pruning cutoff blocks for known networks.
// They point to the first post-merge block. Any pruning should truncate *up to* but excluding
// given block.
var HistoryPrunePoints = map[common.Hash]*HistoryPrunePoint{
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
