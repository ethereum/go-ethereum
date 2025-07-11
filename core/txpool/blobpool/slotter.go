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

// newSlotter creates a helper method for the Billy datastore that returns the
// individual shelf sizes used to store transactions in.
//
// The slotter will create shelves for each possible blob count + some tx metadata
// wiggle room, up to the max permitted limits.
//
// The slotter also creates a shelf for 0-blob transactions. Whilst those are not
// allowed in the current protocol, having an empty shelf is not a relevant use
// of resources, but it makes stress testing with junk transactions simpler.
func newSlotter(maxBlobsPerTransaction int) func() (uint32, bool) {
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
func newSlotterEIP7594(maxBlobsPerTransaction int) func() (uint32, bool) {
	slotsize := uint32(txAvgSize)
	slotsize -= uint32(blobSize) + txBlobOverhead // underflows, it's ok, will overflow back in the first return

	return func() (size uint32, done bool) {
		slotsize += blobSize + txBlobOverhead
		finished := slotsize > uint32(maxBlobsPerTransaction)*blobSize+txMaxSize

		return slotsize, finished
	}
}
