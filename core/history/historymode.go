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
	case "postosaka":
		*m = KeepPostOsaka
	case "custom":
		*m = KeepCustom
	default:
		return fmt.Errorf(`unknown history mode %q, want "all", "postmerge", "postprague", "postosaka", or "custom"`, text)
	}
	return nil
}

// HistoryModeNames returns the accepted values for a history mode, for use in
// flag usage strings and error messages.
func HistoryModeNames() []string {
	return []string{"all", "postmerge", "postprague", "postosaka", "custom"}
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

// ParsePrunePoint parses a history pruning point given as "<number>:<hash>", the
// form accepted by the --history.chain flag.
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

// ChainHistoryNames lists the named values of the --history.chain flag.
func ChainHistoryNames() string {
	var names []string
	for _, name := range HistoryModeNames() {
		if name != KeepCustom.String() {
			names = append(names, name)
		}
	}
	return `"` + strings.Join(names, `", "`) + `"`
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
}

// HistoryPolicy describes the configured history pruning strategy. It captures
// user intent as opposed to the actual DB state.
type HistoryPolicy struct {
	Mode   HistoryMode
	Target *PrunePoint
}

// String implements fmt.Stringer.
func (p HistoryPolicy) String() string {
	if p.Mode == KeepCustom {
		return p.Target.String()
	}
	return p.Mode.String()
}

// MarshalText implements encoding.TextMarshaler.
func (p HistoryPolicy) MarshalText() ([]byte, error) {
	if p.Mode == KeepCustom && p.Target == nil {
		return nil, errors.New("custom history retention without a prune point")
	}
	if !p.Mode.IsValid() {
		return nil, fmt.Errorf("unknown history mode %d", p.Mode)
	}
	return []byte(p.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (p *HistoryPolicy) UnmarshalText(text []byte) error {
	var (
		value = string(text)
		mode  HistoryMode
	)
	if err := mode.UnmarshalText(text); err == nil {
		if mode == KeepCustom {
			return errors.New(`custom history retention is given as "<block number>:<block hash>"`)
		}
		*p = HistoryPolicy{Mode: mode}
		return nil
	}
	if !strings.Contains(value, ":") {
		return fmt.Errorf(`unknown history retention %q, want %s, or "<block number>:<block hash>"`, value, ChainHistoryNames())
	}
	point, err := ParsePrunePoint(value)
	if err != nil {
		return err
	}
	*p = HistoryPolicy{
		Mode:   KeepCustom,
		Target: point,
	}
	return nil
}

// Resolve returns the policy for the network with the given genesis hash,
// filling in the built-in prune point for the fork-based modes.
//
// A prune point is only accepted for KeepCustom, or for a fork-based mode when it
// equals the built-in one, so that resolving an already resolved policy works.
func (p HistoryPolicy) Resolve(genesisHash common.Hash) (HistoryPolicy, error) {
	switch p.Mode {
	case KeepAll:
		if p.Target != nil {
			return HistoryPolicy{}, fmt.Errorf("history mode %q does not take a prune point", p.Mode)
		}
		return HistoryPolicy{Mode: KeepAll}, nil

	case KeepPostMerge, KeepPostPrague, KeepPostOsaka:
		point := staticPrunePoints[p.Mode][genesisHash]
		if point == nil {
			return HistoryPolicy{}, fmt.Errorf("%s history pruning not available for network %s", p.Mode, genesisHash.Hex())
		}
		if p.Target != nil && *p.Target != *point {
			return HistoryPolicy{}, fmt.Errorf("history mode %q prunes to %s, not %s", p.Mode, point, p.Target)
		}
		return HistoryPolicy{Mode: p.Mode, Target: point}, nil

	case KeepCustom:
		if p.Target == nil {
			return HistoryPolicy{}, fmt.Errorf("history mode %q requires a prune point, given as \"<block number>:<block hash>\"", p.Mode)
		}
		if p.Target.BlockNumber == 0 {
			return HistoryPolicy{}, fmt.Errorf("history mode %q: prune point must be above the genesis block", p.Mode)
		}
		if p.Target.BlockHash == (common.Hash{}) {
			return HistoryPolicy{}, fmt.Errorf("history mode %q: prune point block hash must not be zero", p.Mode)
		}
		return p, nil

	default:
		return HistoryPolicy{}, fmt.Errorf("invalid history mode: %d", p.Mode)
	}
}

// PrunedHistoryError is returned by APIs when the requested history is pruned.
type PrunedHistoryError struct{}

func (e *PrunedHistoryError) Error() string  { return "pruned history unavailable" }
func (e *PrunedHistoryError) ErrorCode() int { return 4444 }
