// Copyright 2017 The go-ethereum Authors
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

// +build go1.8

package ethash

import "math/big"

// cacheSize calculates and returns the size of the ethash verification cache that
// belongs to a certain block number. The cache size grows linearly, however, we
// always take the highest prime below the linearly growing threshold in order to
// reduce the risk of accidental regularities leading to cyclic behavior.
func cacheSize(block uint64) uint64 {
	// If we have a pre-generated value, use that
	epoch := int(block / epochLength)
	if epoch < len(cacheSizes) {
		return cacheSizes[epoch]
	}
	// No known cache size, calculate manually (sanity branch only)
	size := cacheInitBytes + cacheGrowthBytes*uint64(epoch) - hashBytes
	for !new(big.Int).SetUint64(size / hashBytes).ProbablyPrime(1) { // Always accurate for n < 2^64
		size -= 2 * hashBytes
	}
	return size
}

// datasetSize calculates and returns the size of the ethash mining dataset that
// belongs to a certain block number. The dataset size grows linearly, however, we
// always take the highest prime below the linearly growing threshold in order to
// reduce the risk of accidental regularities leading to cyclic behavior.
func datasetSize(block uint64) uint64 {
	// If we have a pre-generated value, use that
	epoch := int(block / epochLength)
	if epoch < len(datasetSizes) {
		return datasetSizes[epoch]
	}
	// No known dataset size, calculate manually (sanity branch only)
	size := datasetInitBytes + datasetGrowthBytes*uint64(epoch) - mixBytes
	for !new(big.Int).SetUint64(size / mixBytes).ProbablyPrime(1) { // Always accurate for n < 2^64
		size -= 2 * mixBytes
	}
	return size
}
