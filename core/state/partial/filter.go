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

import "github.com/ethereum/go-ethereum/common"

// ContractFilter determines which contracts' storage to sync and retain.
// This interface allows flexible filtering strategies for partial statefulness.
type ContractFilter interface {
	// ShouldSyncStorage returns true if we should download storage for this contract
	// during snap sync. Returns false for contracts whose storage we skip.
	ShouldSyncStorage(address common.Address) bool

	// ShouldSyncCode returns true if we should download bytecode for this contract
	// during snap sync. Returns false for contracts whose code we skip.
	ShouldSyncCode(address common.Address) bool

	// IsTracked returns true if this contract's storage is being tracked.
	// Used by RPC handlers to determine if storage queries can be answered.
	IsTracked(address common.Address) bool
}

// ConfiguredFilter implements ContractFilter based on a configured list of addresses.
// This is the primary implementation used in production.
type ConfiguredFilter struct {
	contracts map[common.Address]struct{}
}

// NewConfiguredFilter creates a new filter from a list of contract addresses.
func NewConfiguredFilter(addresses []common.Address) *ConfiguredFilter {
	m := make(map[common.Address]struct{}, len(addresses))
	for _, addr := range addresses {
		m[addr] = struct{}{}
	}
	return &ConfiguredFilter{contracts: m}
}

// ShouldSyncStorage returns true if the contract is in the configured list.
func (f *ConfiguredFilter) ShouldSyncStorage(addr common.Address) bool {
	_, ok := f.contracts[addr]
	return ok
}

// ShouldSyncCode returns true if the contract is in the configured list.
func (f *ConfiguredFilter) ShouldSyncCode(addr common.Address) bool {
	_, ok := f.contracts[addr]
	return ok
}

// IsTracked returns true if the contract is in the configured list.
func (f *ConfiguredFilter) IsTracked(addr common.Address) bool {
	_, ok := f.contracts[addr]
	return ok
}

// Contracts returns the list of tracked contract addresses.
func (f *ConfiguredFilter) Contracts() []common.Address {
	result := make([]common.Address, 0, len(f.contracts))
	for addr := range f.contracts {
		result = append(result, addr)
	}
	return result
}

// AllowAllFilter is a filter that allows all contracts (full node behavior).
// Used when partial state mode is disabled.
type AllowAllFilter struct{}

// ShouldSyncStorage always returns true for full node behavior.
func (f *AllowAllFilter) ShouldSyncStorage(addr common.Address) bool {
	return true
}

// ShouldSyncCode always returns true for full node behavior.
func (f *AllowAllFilter) ShouldSyncCode(addr common.Address) bool {
	return true
}

// IsTracked always returns true for full node behavior.
func (f *AllowAllFilter) IsTracked(addr common.Address) bool {
	return true
}
