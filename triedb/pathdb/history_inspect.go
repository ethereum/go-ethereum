// Copyright 2024 The go-ethereum Authors
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

package pathdb

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// HistoryStats wraps the history inspection statistics.
type HistoryStats struct {
	Start   uint64   // Block number of the first queried history
	End     uint64   // Block number of the last queried history
	Blocks  []uint64 // Blocks refers to the list of block numbers in which the state is mutated
	Origins [][]byte // Origins refers to the original value of the state before its mutation
}

// sanitizeRange limits the given range to fit within the local history store.
func sanitizeRange(start, end uint64, freezer ethdb.AncientReader) (uint64, uint64, error) {
	// Load the id of the first history object in local store.
	tail, err := freezer.Tail()
	if err != nil {
		return 0, 0, err
	}
	first := tail + 1
	if start != 0 && start > first {
		first = start
	}
	// Load the id of the last history object in local store.
	head, err := freezer.Ancients()
	if err != nil {
		return 0, 0, err
	}
	last := head - 1
	if end != 0 && end < last {
		last = end
	}
	// Make sure the range is valid
	if first >= last {
		return 0, 0, fmt.Errorf("range is invalid, first: %d, last: %d", first, last)
	}
	return first, last, nil
}

func inspectHistory(freezer ethdb.AncientReader, start, end uint64, onHistory func(*history, *HistoryStats)) (*HistoryStats, error) {
	var (
		stats  = &HistoryStats{}
		init   = time.Now()
		logged = time.Now()
	)
	start, end, err := sanitizeRange(start, end, freezer)
	if err != nil {
		return nil, err
	}
	for id := start; id <= end; id += 1 {
		// The entire history object is decoded, although it's unnecessary for
		// account inspection. TODO(rjl493456442) optimization is worthwhile.
		h, err := readHistory(freezer, id)
		if err != nil {
			return nil, err
		}
		if id == start {
			stats.Start = h.meta.block
		}
		if id == end {
			stats.End = h.meta.block
		}
		onHistory(h, stats)

		if time.Since(logged) > time.Second*8 {
			logged = time.Now()
			eta := float64(time.Since(init)) / float64(id-start+1) * float64(end-id)
			log.Info("Inspecting state history", "checked", id-start+1, "left", end-id, "elapsed", common.PrettyDuration(time.Since(init)), "eta", common.PrettyDuration(eta))
		}
	}
	log.Info("Inspected state history", "total", end-start+1, "elapsed", common.PrettyDuration(time.Since(init)))
	return stats, nil
}

// accountHistory inspects the account history within the range.
func accountHistory(freezer ethdb.AncientReader, address common.Address, start, end uint64) (*HistoryStats, error) {
	return inspectHistory(freezer, start, end, func(h *history, stats *HistoryStats) {
		blob, exists := h.accounts[address]
		if !exists {
			return
		}
		stats.Blocks = append(stats.Blocks, h.meta.block)
		stats.Origins = append(stats.Origins, blob)
	})
}

// storageHistory inspects the storage history within the range.
func storageHistory(freezer ethdb.AncientReader, address common.Address, slot common.Hash, start uint64, end uint64) (*HistoryStats, error) {
	slotHash := crypto.Keccak256Hash(slot.Bytes())
	return inspectHistory(freezer, start, end, func(h *history, stats *HistoryStats) {
		slots, exists := h.storages[address]
		if !exists {
			return
		}
		key := slotHash
		if h.meta.version != stateHistoryV0 {
			key = slot
		}
		blob, exists := slots[key]
		if !exists {
			return
		}
		stats.Blocks = append(stats.Blocks, h.meta.block)
		stats.Origins = append(stats.Origins, blob)
	})
}

// historyRange returns the block number range of local state histories.
func historyRange(freezer ethdb.AncientReader) (uint64, uint64, error) {
	// Load the id of the first history object in local store.
	tail, err := freezer.Tail()
	if err != nil {
		return 0, 0, err
	}
	first := tail + 1

	// Load the id of the last history object in local store.
	head, err := freezer.Ancients()
	if err != nil {
		return 0, 0, err
	}
	last := head - 1

	fh, err := readHistory(freezer, first)
	if err != nil {
		return 0, 0, err
	}
	lh, err := readHistory(freezer, last)
	if err != nil {
		return 0, 0, err
	}
	return fh.meta.block, lh.meta.block, nil
}
