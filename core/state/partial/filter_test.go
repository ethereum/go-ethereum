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

package partial

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestConfiguredFilterBasic(t *testing.T) {
	// Test empty filter
	emptyFilter := NewConfiguredFilter(nil)
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	if emptyFilter.ShouldSyncStorage(addr) {
		t.Error("Empty filter should not allow any storage")
	}
	if emptyFilter.ShouldSyncCode(addr) {
		t.Error("Empty filter should not allow any code")
	}
	if emptyFilter.IsTracked(addr) {
		t.Error("Empty filter should not track any address")
	}

	// Test filter with addresses
	tracked := []common.Address{
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
		common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
	}
	filter := NewConfiguredFilter(tracked)

	// Tracked addresses should pass
	for _, addr := range tracked {
		if !filter.ShouldSyncStorage(addr) {
			t.Errorf("Tracked address %s should allow storage", addr.Hex())
		}
	}

	// Untracked address should not pass
	untracked := common.HexToAddress("0x0000000000000000000000000000000000000001")
	if filter.ShouldSyncStorage(untracked) {
		t.Error("Untracked address should not allow storage")
	}
}

func TestConfiguredFilterHashConsistency(t *testing.T) {
	tracked := []common.Address{
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
	}
	filter := NewConfiguredFilter(tracked)

	// Address-based and hash-based methods should be consistent
	for _, addr := range tracked {
		hash := crypto.Keccak256Hash(addr.Bytes())

		addrStorage := filter.ShouldSyncStorage(addr)
		hashStorage := filter.ShouldSyncStorageByHash(hash)
		if addrStorage != hashStorage {
			t.Errorf("Inconsistent storage filter: addr=%v, hash=%v", addrStorage, hashStorage)
		}

		addrCode := filter.ShouldSyncCode(addr)
		hashCode := filter.ShouldSyncCodeByHash(hash)
		if addrCode != hashCode {
			t.Errorf("Inconsistent code filter: addr=%v, hash=%v", addrCode, hashCode)
		}
	}
}

func TestAllowAllFilterInterface(t *testing.T) {
	// Verify AllowAllFilter implements ContractFilter
	var filter ContractFilter = &AllowAllFilter{}

	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	hash := crypto.Keccak256Hash(addr.Bytes())

	if !filter.ShouldSyncStorage(addr) {
		t.Error("AllowAllFilter should allow storage")
	}
	if !filter.ShouldSyncCode(addr) {
		t.Error("AllowAllFilter should allow code")
	}
	if !filter.IsTracked(addr) {
		t.Error("AllowAllFilter should track all addresses")
	}
	if !filter.ShouldSyncStorageByHash(hash) {
		t.Error("AllowAllFilter should allow storage by hash")
	}
	if !filter.ShouldSyncCodeByHash(hash) {
		t.Error("AllowAllFilter should allow code by hash")
	}
}
