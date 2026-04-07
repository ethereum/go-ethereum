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

package ethapi

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	corestate "github.com/ethereum/go-ethereum/core/state"
)

// HistoryRetention reports a node's configured history retention windows.
// It is consumed by the eth_capabilities RPC method to derive the response
// described in https://github.com/ethereum/execution-apis/pull/755.
type HistoryRetention struct {
	// TxIndexHistory is the number of recent blocks for which the
	// transaction lookup index is maintained. Zero means the index covers
	// the entire available chain.
	TxIndexHistory uint64

	// LogIndexHistory is the number of recent blocks for which the log
	// search index is maintained. Zero means the index covers the entire
	// available chain.
	LogIndexHistory uint64

	// LogIndexDisabled reports whether the log search index has been
	// turned off entirely.
	LogIndexDisabled bool

	// StateHistory is the number of recent blocks for which historical
	// state is retained in path-based archive mode. Zero means the entire
	// available state history is kept.
	StateHistory uint64

	// TrienodeHistory is the number of recent blocks for which trie node
	// history is retained in path-based archive mode. Zero means the entire
	// available trienode history is kept; negative means no trienode history
	// is stored.
	TrienodeHistory int64

	// StateArchive reports whether state pruning is disabled
	// (--gcmode=archive).
	StateArchive bool

	// StateScheme is the state storage scheme in use, either "hash" or
	// "path".
	StateScheme string
}

// Capabilities reports which historical data the node can serve. It is
// returned by the eth_capabilities RPC method as defined in
// https://github.com/ethereum/execution-apis/pull/755.
type Capabilities struct {
	Head        CapabilityHead     `json:"head"`
	State       CapabilityResource `json:"state"`
	Tx          CapabilityResource `json:"tx"`
	Logs        CapabilityResource `json:"logs"`
	Receipts    CapabilityResource `json:"receipts"`
	Blocks      CapabilityResource `json:"blocks"`
	StateProofs CapabilityResource `json:"stateproofs"`
}

// CapabilityHead is the current canonical head as reported by the node.
type CapabilityHead struct {
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	BlockHash   common.Hash    `json:"blockHash"`
}

// CapabilityResource describes the availability of a single data resource.
type CapabilityResource struct {
	Disabled       bool           `json:"disabled"`
	OldestBlock    hexutil.Uint64 `json:"oldestBlock"`
	DeleteStrategy DeleteStrategy `json:"deleteStrategy"`
}

// DeleteStrategy describes how data of a resource is removed over time.
//
// Two strategies are defined by the spec:
//
//   - "none":   data is never deleted; the resource is permanently
//     retained from oldestBlock onwards.
//   - "window": data is retained for a sliding window of the most recent
//     RetentionBlocks blocks.
//
// RetentionBlocks is omitted from the JSON output for the "none" strategy.
type DeleteStrategy struct {
	Type            string  `json:"type"`
	RetentionBlocks *uint64 `json:"retentionBlocks,omitempty"`
}

// strategyNone returns a DeleteStrategy with type "none".
func strategyNone() DeleteStrategy {
	return DeleteStrategy{Type: "none"}
}

// strategyWindow returns a DeleteStrategy with type "window" and the given
// retention block count.
func strategyWindow(retention uint64) DeleteStrategy {
	return DeleteStrategy{Type: "window", RetentionBlocks: &retention}
}

// Capabilities implements the eth_capabilities RPC method as defined in
// https://github.com/ethereum/execution-apis/pull/755. It returns a
// description of the historical data this node can serve, allowing RPC
// routers to determine which queries can be answered without hitting
// "history pruned" errors.
func (api *BlockChainAPI) Capabilities() *Capabilities {
	head := api.b.CurrentHeader()
	return buildCapabilities(
		head.Number.Uint64(),
		head.Hash(),
		api.b.HistoryPruningCutoff(),
		api.b.HistoryRetention(),
	)
}

// buildCapabilities computes the eth_capabilities response from the head
// block, the absolute history pruning cutoff, and the configured retention
// windows. It is split out from the RPC method so the mapping rules can be
// unit tested without a backend.
func buildCapabilities(headNum uint64, headHash common.Hash, cutoff uint64, ret HistoryRetention) *Capabilities {
	// windowOldest returns the oldest block reachable through a sliding
	// window of `window` blocks, never going below the supplied floor. A
	// window of zero means "no sliding deletion" and reports the floor
	// itself.
	windowOldest := func(window uint64, floor uint64) uint64 {
		if window == 0 || headNum+1 <= window {
			return floor
		}
		oldest := headNum + 1 - window
		if oldest < floor {
			return floor
		}
		return oldest
	}

	// resource builds a CapabilityResource for a window-style resource.
	// A window of zero is reported as deleteStrategy "none".
	resource := func(disabled bool, window uint64, floor uint64) CapabilityResource {
		ds := strategyNone()
		if window != 0 {
			ds = strategyWindow(window)
		}
		return CapabilityResource{
			Disabled:       disabled,
			OldestBlock:    hexutil.Uint64(windowOldest(window, floor)),
			DeleteStrategy: ds,
		}
	}

	// Bodies and receipts share the same retention model in
	// geth: they are either kept in full ("all") or pruned to a fixed
	// boundary ("postmerge"). In neither case is there a sliding
	// deletion window, so the strategy is always "none" and the oldest
	// block equals the history pruning cutoff.
	blocks := CapabilityResource{
		Disabled:       false,
		OldestBlock:    hexutil.Uint64(cutoff),
		DeleteStrategy: strategyNone(),
	}
	receipts := blocks

	tx := resource(false, ret.TxIndexHistory, cutoff)
	logs := resource(ret.LogIndexDisabled, ret.LogIndexHistory, cutoff)

	// State availability is determined primarily by gcmode:
	//
	//   - full mode:    only the in-memory state window is reachable,
	//                   regardless of the storage scheme.
	//   - archive+hash: full state history is reachable.
	//   - archive+path: honors the configured StateHistory window.
	var state CapabilityResource
	switch {
	case !ret.StateArchive:
		state = resource(false, corestate.TriesInMemory, 0)
	case ret.StateScheme == rawdb.HashScheme:
		state = resource(false, 0, 0)
	default:
		state = resource(false, ret.StateHistory, 0)
	}

	// eth_getProof availability tracks state availability in hash mode and
	// in path-based full mode. Path-based archive nodes store trie node
	// history separately from state history.
	stateproofs := state
	if ret.StateArchive && ret.StateScheme == rawdb.PathScheme {
		switch {
		case ret.TrienodeHistory < 0:
			stateproofs = resource(false, corestate.TriesInMemory, 0)
		case ret.TrienodeHistory == 0:
			stateproofs = resource(false, 0, 0)
		default:
			stateproofs = resource(false, uint64(ret.TrienodeHistory), 0)
		}
	}

	return &Capabilities{
		Head: CapabilityHead{
			BlockNumber: hexutil.Uint64(headNum),
			BlockHash:   headHash,
		},
		State:       state,
		Tx:          tx,
		Logs:        logs,
		Receipts:    receipts,
		Blocks:      blocks,
		StateProofs: stateproofs,
	}
}
