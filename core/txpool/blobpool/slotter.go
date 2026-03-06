// Copyright 2023 The go-ethereum Authors
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

package blobpool

import (
	"errors"

	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/billy"
)

// slotSizer computes the storage shelf size for a given transaction size using
// O(1) arithmetic. Shelf sizes form an arithmetic sequence:
//
//	base, base+step, base+2*step, ...
//
// This mirrors the progression in newSlotter and newSlotterEIP7594, but avoids
// creating and iterating a stateful closure on every lookup.
type slotSizer struct {
	base uint32 // Size of the first shelf (txAvgSize)
	step uint32 // Size increment per subsequent shelf
	max  uint32 // Largest valid shelf size
}

// newSlotSizer creates a slotSizer by consuming a slotter closure once to
// discover its base size, step size, and maximum shelf size.
func newSlotSizer(slotter billy.SlotSizeFn) slotSizer {
	first, done := slotter()
	if done {
		return slotSizer{base: first, step: 0, max: first}
	}
	second, done := slotter()
	step := second - first
	last := second
	for !done {
		last, done = slotter()
	}
	return slotSizer{base: first, step: step, max: last}
}

// getSlotSize returns the shelf size that can store a transaction of the given
// byte size, or an error if it exceeds the largest shelf.
func (s slotSizer) getSlotSize(size uint32) (uint32, error) {
	if size <= s.base {
		return s.base, nil
	}
	// Round up to the nearest shelf: base + ⌈(size-base)/step⌉ * step
	slot := s.base + ((size-s.base+s.step-1)/s.step)*s.step
	if slot > s.max {
		return 0, errors.New("size exceeds maximum slot size")
	}
	return slot, nil
}

// tryMigrate checks if the billy needs to be migrated and migrates if needed.
// Returns a slotter that can be used for the database.
func tryMigrate(config *params.ChainConfig, slotter billy.SlotSizeFn, datadir string) (billy.SlotSizeFn, error) {
	// Check if we need to migrate our blob db to the new slotter.
	if config.OsakaTime != nil {
		// Open the store using the version slotter to see if any version has been
		// written.
		var version int
		index := func(_ uint64, _ uint32, blob []byte) {
			version = max(version, parseSlotterVersion(blob))
		}
		store, err := billy.Open(billy.Options{Path: datadir}, newVersionSlotter(), index)
		if err != nil {
			return nil, err
		}
		store.Close()

		// If the version found is less than the currently configured store version,
		// perform a migration then write the updated version of the store.
		if version < storeVersion {
			newSlotter := newSlotterEIP7594(params.BlobTxMaxBlobs)
			if err := billy.Migrate(billy.Options{Path: datadir, Repair: true}, slotter, newSlotter); err != nil {
				return nil, err
			}
			store, err = billy.Open(billy.Options{Path: datadir}, newVersionSlotter(), nil)
			if err != nil {
				return nil, err
			}
			writeSlotterVersion(store, storeVersion)
			store.Close()
		}
		// Set the slotter to the format now that the Osaka is active.
		slotter = newSlotterEIP7594(params.BlobTxMaxBlobs)
	}
	return slotter, nil
}

// newSlotter creates a helper method for the Billy datastore that returns the
// individual shelf sizes used to store transactions in.
//
// The slotter will create shelves for each possible blob count + some tx metadata
// wiggle room, up to the max permitted limits.
//
// The slotter also creates a shelf for 0-blob transactions. Whilst those are not
// allowed in the current protocol, having an empty shelf is not a relevant use
// of resources, but it makes stress testing with junk transactions simpler.
func newSlotter(maxBlobsPerTransaction int) billy.SlotSizeFn {
	slotsize := uint32(txAvgSize)
	slotsize -= uint32(blobSize) // underflows, it's ok, will overflow back in the first return

	return func() (size uint32, done bool) {
		slotsize += blobSize
		finished := slotsize > uint32(maxBlobsPerTransaction)*blobSize+txMaxSize

		return slotsize, finished
	}
}

// newSlotterEIP7594 creates a different slotter for EIP-7594 transactions.
// EIP-7594 (PeerDAS) changes the average transaction size which means the current
// static 4KB average size is not enough anymore.
// This slotter adds a dynamic overhead component to the slotter, which also
// captures the notion that blob transactions with more blobs are also more likely to
// to have more calldata.
func newSlotterEIP7594(maxBlobsPerTransaction int) billy.SlotSizeFn {
	slotsize := uint32(txAvgSize)
	slotsize -= uint32(blobSize) + txBlobOverhead // underflows, it's ok, will overflow back in the first return

	return func() (size uint32, done bool) {
		slotsize += blobSize + txBlobOverhead
		finished := slotsize > uint32(maxBlobsPerTransaction)*(blobSize+txBlobOverhead)+txMaxSize

		return slotsize, finished
	}
}

// newVersionSlotter creates a slotter with a single 8 byte shelf to store
// version metadata in.
func newVersionSlotter() billy.SlotSizeFn {
	return func() (size uint32, done bool) {
		return 8, true
	}
}

// parseSlotterVersion will parse the slotter's version from a given data blob.
func parseSlotterVersion(blob []byte) int {
	if len(blob) > 0 {
		return int(blob[0])
	}
	return 0
}

// writeSlotterVersion writes the current slotter version into the store.
func writeSlotterVersion(store billy.Database, version int) {
	store.Put([]byte{byte(version)})
}
