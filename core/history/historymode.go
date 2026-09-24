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
	"errors"
	"fmt"
	"strconv"
	"strings"

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

	// KeepPostOsaka sets the history pruning point to the Osaka activation block.
	KeepPostOsaka

	// KeepPostMay2026 sets the history pruning point to a fixed date instead of a
	// fork transition. Such a point lets a node prune to a chosen date without
	// waiting for the next fork; see staticPrunePoints for the derivation.
	KeepPostMay2026

	// KeepCustom sets the history pruning point to a block number and hash supplied
	// by the operator, in place of one of the built-in points.
	KeepCustom
)

func (m HistoryMode) IsValid() bool {
	return m <= KeepCustom
}

func (m HistoryMode) String() string {
	switch m {
	case KeepAll:
		return "all"
	case KeepPostMerge:
		return "postmerge"
	case KeepPostPrague:
		return "postprague"
	case KeepPostOsaka:
		return "postosaka"
	case KeepPostMay2026:
		return "2026-05"
	case KeepCustom:
		return "custom"
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
	case "postosaka", "osaka":
		*m = KeepPostOsaka
	case "2026-05", "post2026-05":
		*m = KeepPostMay2026
	case "custom":
		*m = KeepCustom
	default:
		return fmt.Errorf(`unknown history mode %q, want "all", "postmerge", "postprague", "postosaka", "2026-05", or "custom"`, text)
	}
	return nil
}

// HistoryModeNames returns the accepted values for a history mode, for use in
// flag usage strings and error messages.
func HistoryModeNames() []string {
	return []string{"all", "postmerge", "postprague", "postosaka", "2026-05", "custom"}
}

// PrunePoint identifies a specific block for history pruning.
type PrunePoint struct {
	BlockNumber uint64
	BlockHash   common.Hash
}

// String formats the prune point in the "number:hash" form that ParsePrunePoint
// accepts, so a configured point can be echoed back as a command-line argument.
func (p *PrunePoint) String() string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d:%s", p.BlockNumber, p.BlockHash.Hex())
}

// MarshalText implements encoding.TextMarshaler, and UnmarshalText the counterpart.
// They make a configured prune point serialise as the same "number:hash" string the
// --history.tail flag takes. Keeping it a string rather than a nested table also
// matters for the TOML config file, where a table cannot be followed by plain keys.
//
// Both are on the pointer receiver: the TOML library reflects on *PrunePoint and calls
// the marshaller even when the pointer is nil, which a value receiver would turn into
// a nil dereference.
func (p *PrunePoint) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *PrunePoint) UnmarshalText(text []byte) error {
	point, err := ParsePrunePoint(string(text))
	if err != nil {
		return err
	}
	*p = *point
	return nil
}

// ParsePrunePoint parses a history pruning point given as "<number>:<hash>", the
// form accepted by the --history.tail flag.
//
// Both halves are required. The number alone cannot be trusted as a pruning point
// because nothing guarantees the canonical chain includes it (a reorged block has
// a number too); the hash alone says nothing about where to cut. The pair is
// checked against the canonical chain when pruning runs.
func ParsePrunePoint(input string) (*PrunePoint, error) {
	number, hash, ok := strings.Cut(input, ":")
	if !ok {
		return nil, fmt.Errorf(`invalid prune point %q, want "<block number>:<block hash>"`, input)
	}
	block, err := strconv.ParseUint(number, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid block number %q in prune point: %v", number, err)
	}
	if block == 0 {
		return nil, errors.New("prune point must be above the genesis block")
	}
	var point common.Hash
	if err := point.UnmarshalText([]byte(hash)); err != nil {
		return nil, fmt.Errorf("invalid block hash %q in prune point: %v", hash, err)
	}
	if point == (common.Hash{}) {
		return nil, errors.New("prune point block hash must not be zero")
	}
	return &PrunePoint{BlockNumber: block, BlockHash: point}, nil
}

// staticPrunePoints contains the pre-defined history pruning cutoff blocks for
// known networks, keyed by history mode and genesis hash. They point to the first
// block after the respective fork. Any pruning should truncate *up to* but
// excluding the given block.
var staticPrunePoints = map[HistoryMode]map[common.Hash]*PrunePoint{
	KeepPostMerge: {
		params.MainnetGenesisHash: {
			BlockNumber: 15537393,
			BlockHash:   common.HexToHash("0x55b11b918355b1ef9c5db810302ebad0bf2544255b530cdce90674d5887bb286"),
		},
		params.SepoliaGenesisHash: {
			BlockNumber: 1450409,
			BlockHash:   common.HexToHash("0x229f6b18ca1552f1d5146deceb5387333f40dc6275aebee3f2c5c4ece07d02db"),
		},
	},
	KeepPostPrague: {
		params.MainnetGenesisHash: {
			BlockNumber: 22431084,
			BlockHash:   common.HexToHash("0x50c8cab760b2948349c590461b166773c45d8f4858cccf5a43025ab2960152e8"),
		},
		params.SepoliaGenesisHash: {
			BlockNumber: 7836331,
			BlockHash:   common.HexToHash("0xe6571beb68bf24dbd8a6ba354518996920c55a3f8d8fdca423e391b8ad071f22"),
		},
		params.HoodiGenesisHash: {
			BlockNumber: 60412,
			BlockHash:   common.HexToHash("0x1562792812ef418eaafc8f1f093d84d9634971e9dd6b0771302eb5b9fd4d2c46"),
		},
	},
	KeepPostOsaka: {
		params.MainnetGenesisHash: {
			BlockNumber: 23935694,
			BlockHash:   common.HexToHash("0x8281db4990564caca17e620a70847c7b001ad70cd7bb5e64aa78adea16747677"),
		},
		params.SepoliaGenesisHash: {
			BlockNumber: 9408577,
			BlockHash:   common.HexToHash("0xae5366ffdd520588a2a1c01e179c127903eac47e781602368ff88308a5583208"),
		},
		params.HoodiGenesisHash: {
			BlockNumber: 1507304,
			BlockHash:   common.HexToHash("0xff88199acc4b4e29b6129b2fe691e4c84c1bfde3b75727eba2ec2dde89847d24"),
		},
	},
	KeepPostMay2026: {
		// Block 25182208 is the first mainnet block whose timestamp is at or after
		// 2026-05-26T21:33:11Z, and is also an era1 boundary (3074*8192), which keeps a
		// pruned database aligned with the era files that could restore what was removed.
		//
		// Only mainnet is listed. Derive a network's own point the same way - ask a node
		// that still has the history for the header of the first block at or after the
		// chosen instant, by number and hash - and add it here rather than reusing
		// mainnet's pair, whose number need not be canonical elsewhere.
		params.MainnetGenesisHash: {
			BlockNumber: 25182208,
			BlockHash:   common.HexToHash("0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad"),
		},
	},
}

// HistoryPolicy describes the configured history pruning strategy. It captures
// user intent as opposed to the actual DB state.
type HistoryPolicy struct {
	Mode HistoryMode
	// Static prune point for the modes with a fixed target, and the operator's own
	// point for KeepCustom. Nil for KeepAll.
	Target *PrunePoint
}

// NewPolicy constructs a HistoryPolicy from the given mode and genesis hash.
//
// The custom point must be non-nil if and only if mode is KeepCustom: a point
// given for another mode would be silently ignored, which is likelier a mistake
// than a decision. Callers that do not use KeepCustom pass nil.
func NewPolicy(mode HistoryMode, genesisHash common.Hash, custom *PrunePoint) (HistoryPolicy, error) {
	if mode != KeepCustom && custom != nil {
		return HistoryPolicy{}, fmt.Errorf("history mode %q does not take a custom prune point, use %q", mode, KeepCustom)
	}
	switch mode {
	case KeepAll:
		return HistoryPolicy{Mode: KeepAll}, nil

	case KeepPostMerge, KeepPostPrague, KeepPostOsaka, KeepPostMay2026:
		point := staticPrunePoints[mode][genesisHash]
		if point == nil {
			return HistoryPolicy{}, fmt.Errorf("%s history pruning not available for network %s", mode, genesisHash.Hex())
		}
		return HistoryPolicy{Mode: mode, Target: point}, nil

	case KeepCustom:
		if custom == nil {
			return HistoryPolicy{}, fmt.Errorf("history mode %q requires a prune point, given as \"<block number>:<block hash>\"", mode)
		}
		if custom.BlockNumber == 0 {
			return HistoryPolicy{}, fmt.Errorf("history mode %q: prune point must be above the genesis block", mode)
		}
		if custom.BlockHash == (common.Hash{}) {
			return HistoryPolicy{}, fmt.Errorf("history mode %q: prune point block hash must not be zero", mode)
		}
		return HistoryPolicy{Mode: mode, Target: custom}, nil

	default:
		return HistoryPolicy{}, fmt.Errorf("invalid history mode: %d", mode)
	}
}

// PrunedHistoryError is returned by APIs when the requested history is pruned.
type PrunedHistoryError struct{}

func (e *PrunedHistoryError) Error() string  { return "pruned history unavailable" }
func (e *PrunedHistoryError) ErrorCode() int { return 4444 }
