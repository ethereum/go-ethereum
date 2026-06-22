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

package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// benchStateReader is a minimal database.StateReader backed by an in-memory map.
// It isolates the conversion overhead in flatReader.Account (address hashing,
// the *types.StateAccount allocation and the slim->full fixups) from any trie,
// snapshot or disk access.
type benchStateReader struct {
	account *types.SlimAccount
}

func (r *benchStateReader) Account(hash common.Hash) (*types.SlimAccount, error) {
	// Return a fresh copy on every call, mirroring the contract that the
	// returned account is safe to modify by the caller.
	if r.account == nil {
		return nil, nil
	}
	cpy := *r.account
	return &cpy, nil
}

func (r *benchStateReader) Storage(accountHash, storageHash common.Hash) ([]byte, error) {
	return nil, nil
}

// benchmarkFlatReaderAccount measures flatReader.Account for a single address.
// The provided slim account dictates which branches of the conversion are hit:
// a slim account with nil Root/CodeHash exercises the EmptyCodeHash.Bytes() and
// EmptyRootHash fixups (the lines that dominate the production profile), whereas
// a fully populated one skips them.
func benchmarkFlatReaderAccount(b *testing.B, slim *types.SlimAccount) {
	r := newFlatReader(&benchStateReader{account: slim})
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		acct, err := r.Account(addr)
		if err != nil {
			b.Fatal(err)
		}
		if acct == nil {
			b.Fatal("unexpected nil account")
		}
	}
}

// BenchmarkFlatReaderAccountEmpty benchmarks the common EOA case: an account
// with no code and no storage. This is the hot path from the profile, hitting
// both the EmptyCodeHash and EmptyRootHash fixups.
func BenchmarkFlatReaderAccountEmpty(b *testing.B) {
	benchmarkFlatReaderAccount(b, &types.SlimAccount{
		Nonce:   1,
		Balance: uint256.NewInt(100),
		// Root and CodeHash left nil: slim encoding of an EOA.
	})
}

// BenchmarkFlatReaderAccountContract benchmarks a contract account with a
// non-empty storage root and code hash, skipping the empty-value fixups.
func BenchmarkFlatReaderAccountContract(b *testing.B) {
	root := common.HexToHash("0xaabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")
	codeHash := common.HexToHash("0x112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00")
	benchmarkFlatReaderAccount(b, &types.SlimAccount{
		Nonce:    7,
		Balance:  uint256.NewInt(1000),
		Root:     root.Bytes(),
		CodeHash: codeHash.Bytes(),
	})
}
