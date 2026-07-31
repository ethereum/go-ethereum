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

package vm

import "testing"

// TestFlatGasPrecompilesNotCacheable guards against reintroducing result
// caching for precompiles that charge a flat gas price while reading only a
// fixed-length prefix of an unbounded input. Caching those lets an attacker pad
// the input up to maxCacheablePrecompileInput to force an expensive,
// always-missing cache-key hash at a fixed, cheap gas cost (and for the bn256
// ops the key hash exceeds the computation even on hits). See the Cacheable
// opt-outs in contracts.go.
func TestFlatGasPrecompilesNotCacheable(t *testing.T) {
	// A maximally padded input: eligible by size, but with trailing bytes that
	// none of these precompiles read.
	padded := make([]byte, maxCacheablePrecompileInput)

	precompiles := []struct {
		name string
		p    PrecompiledContract
	}{
		{"ecrecover", &ecrecover{}},
		{"bn256AddIstanbul", &bn256AddIstanbul{}},
		{"bn256AddByzantium", &bn256AddByzantium{}},
		{"bn256ScalarMulIstanbul", &bn256ScalarMulIstanbul{}},
		{"bn256ScalarMulByzantium", &bn256ScalarMulByzantium{}},
	}
	for _, tc := range precompiles {
		if cacheablePrecompile(tc.p, padded) {
			t.Errorf("%s is cacheable at a %d-byte input; it must opt out to avoid unmetered cache-key hashing",
				tc.name, len(padded))
		}
	}
}
