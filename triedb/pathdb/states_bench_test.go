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
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// BenchmarkStateSetAccount benchmarks the account lookup performance.
func BenchmarkStateSetAccount(b *testing.B) {
	// Create a stateSet with a reasonable number of accounts
	const numAccounts = 1000
	accountData := make(map[common.Hash][]byte, numAccounts)

	// Generate random account hashes and data
	hashes := make([]common.Hash, numAccounts)
	for i := 0; i < numAccounts; i++ {
		var hash common.Hash
		rand.Read(hash[:])
		hashes[i] = hash
		accountData[hash] = make([]byte, 32)
		rand.Read(accountData[hash])
	}

	s := newStates(accountData, nil, false)

	// Prepare test hashes: mix of existing and non-existing
	testHashes := make([]common.Hash, 100)
	for i := 0; i < 50; i++ {
		// Use existing hashes
		testHashes[i] = hashes[i*10]
	}
	for i := 50; i < 100; i++ {
		// Use non-existing hashes
		var hash common.Hash
		rand.Read(hash[:])
		testHashes[i] = hash
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		hash := testHashes[i%len(testHashes)]
		_, _ = s.account(hash)
	}
}
