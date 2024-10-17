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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

// counter helps in tracking items and their corresponding sizes.
type counter struct {
	n    int
	size int
}

// add size to the counter and increase the item counter.
func (c *counter) add(size int) {
	c.n++
	c.size += size
}

// report uploads the cached statistics to meters.
func (c *counter) report(count metrics.Meter, size metrics.Meter) {
	count.Mark(int64(c.n))
	size.Mark(int64(c.size))
}

// StateSetWithOrigin wraps the state set with additional original values of the
// mutated states.
type StateSetWithOrigin struct {
	// AccountOrigin represents the account data before the state transition,
	// corresponding to both the accountData and destructSet. It's keyed by the
	// account address. The nil value means the account was not present before.
	accountOrigin map[common.Address][]byte

	// StorageOrigin represents the storage data before the state transition,
	// corresponding to storageData and deleted slots of destructSet. It's keyed
	// by the account address and slot key hash. The nil value means the slot was
	// not present.
	storageOrigin map[common.Address]map[common.Hash][]byte

	// Memory size of the state data (accountOrigin and storageOrigin)
	size uint64
}

// NewStateSetWithOrigin constructs the state set with the provided data.
func NewStateSetWithOrigin(accountOrigin map[common.Address][]byte, storageOrigin map[common.Address]map[common.Hash][]byte) *StateSetWithOrigin {
	// Don't panic for the lazy callers, initialize the nil maps instead.
	if accountOrigin == nil {
		accountOrigin = make(map[common.Address][]byte)
	}
	if storageOrigin == nil {
		storageOrigin = make(map[common.Address]map[common.Hash][]byte)
	}
	// Count the memory size occupied by the set. Note that each slot key here
	// uses 2*common.HashLength to keep consistent with the calculation method
	// of stateSet.
	var size int
	for _, data := range accountOrigin {
		size += common.HashLength + len(data)
	}
	for _, slots := range storageOrigin {
		for _, data := range slots {
			size += 2*common.HashLength + len(data)
		}
	}
	return &StateSetWithOrigin{
		accountOrigin: accountOrigin,
		storageOrigin: storageOrigin,
		size:          uint64(size),
	}
}

// encode serializes the content of state set into the provided writer.
func (s *StateSetWithOrigin) encode(w io.Writer) error {
	// Encode accounts
	type Accounts struct {
		Addresses []common.Address
		Accounts  [][]byte
	}
	var accounts Accounts
	for address, blob := range s.accountOrigin {
		accounts.Addresses = append(accounts.Addresses, address)
		accounts.Accounts = append(accounts.Accounts, blob)
	}
	if err := rlp.Encode(w, accounts); err != nil {
		return err
	}
	// Encode storages
	type Storage struct {
		Address common.Address
		Keys    []common.Hash
		Blobs   [][]byte
	}
	storages := make([]Storage, 0, len(s.storageOrigin))
	for address, slots := range s.storageOrigin {
		keys := make([]common.Hash, 0, len(slots))
		vals := make([][]byte, 0, len(slots))
		for key, val := range slots {
			keys = append(keys, key)
			vals = append(vals, val)
		}
		storages = append(storages, Storage{Address: address, Keys: keys, Blobs: vals})
	}
	return rlp.Encode(w, storages)
}

// decode deserializes the content from the rlp stream into the state set.
func (s *StateSetWithOrigin) decode(r *rlp.Stream) error {
	// Decode account origin
	type Accounts struct {
		Addresses []common.Address
		Accounts  [][]byte
	}
	var (
		accounts   Accounts
		accountSet = make(map[common.Address][]byte)
	)
	if err := r.Decode(&accounts); err != nil {
		return fmt.Errorf("load diff account origin set: %v", err)
	}
	for i := 0; i < len(accounts.Accounts); i++ {
		accountSet[accounts.Addresses[i]] = accounts.Accounts[i]
	}
	s.accountOrigin = accountSet

	// Decode storage origin
	type Storage struct {
		Address common.Address
		Keys    []common.Hash
		Blobs   [][]byte
	}
	var (
		storages   []Storage
		storageSet = make(map[common.Address]map[common.Hash][]byte)
	)
	if err := r.Decode(&storages); err != nil {
		return fmt.Errorf("load diff storage origin: %v", err)
	}
	for _, storage := range storages {
		storageSet[storage.Address] = make(map[common.Hash][]byte)
		for i := 0; i < len(storage.Keys); i++ {
			storageSet[storage.Address][storage.Keys[i]] = storage.Blobs[i]
		}
	}
	s.storageOrigin = storageSet
	return nil
}
