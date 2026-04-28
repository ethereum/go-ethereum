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

package bintrie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Recorder maintains the inverse of the binary-trie key transform: it captures
// every mutation applied to a BinaryTrie keyed by the original address (and,
// for storage, the original slot key) so the post-state can be rendered as a
// types.GenesisAlloc.
type Recorder struct {
	accounts map[common.Address]*types.Account
}

// NewRecorder returns an empty Recorder.
func NewRecorder() *Recorder {
	return &Recorder{accounts: make(map[common.Address]*types.Account)}
}

// entry returns the existing account entry, or creates a fresh one.
func (r *Recorder) entry(addr common.Address) *types.Account {
	if acc, ok := r.accounts[addr]; ok {
		return acc
	}
	acc := &types.Account{}
	r.accounts[addr] = acc
	return acc
}

// RecordAccount upserts the nonce and balance for addr. Existing storage and
// code on the entry are preserved.
func (r *Recorder) RecordAccount(addr common.Address, acc *types.StateAccount) {
	e := r.entry(addr)
	e.Nonce = acc.Nonce
	if acc.Balance != nil {
		e.Balance = acc.Balance.ToBig()
	} else {
		e.Balance = nil
	}
}

// RecordStorage records a storage write. A zero value removes the slot.
func (r *Recorder) RecordStorage(addr common.Address, key, value []byte) {
	k := bytesToHash(key)
	v := bytesToHash(value)
	e := r.entry(addr)
	if (v == common.Hash{}) {
		if e.Storage != nil {
			delete(e.Storage, k)
			if len(e.Storage) == 0 {
				e.Storage = nil
			}
		}
		return
	}
	if e.Storage == nil {
		e.Storage = make(map[common.Hash]common.Hash)
	}
	e.Storage[k] = v
}

// RecordCode records the contract code for addr. Empty code clears the field.
func (r *Recorder) RecordCode(addr common.Address, code []byte) {
	e := r.entry(addr)
	if len(code) == 0 {
		e.Code = nil
		return
	}
	e.Code = common.CopyBytes(code)
}

// RecordDeleteAccount drops addr entirely from the recorded set.
func (r *Recorder) RecordDeleteAccount(addr common.Address) {
	delete(r.accounts, addr)
}

// RecordDeleteStorage clears a single storage slot for addr.
func (r *Recorder) RecordDeleteStorage(addr common.Address, key []byte) {
	r.RecordStorage(addr, key, nil)
}

// Alloc returns the recorded post-state as a types.GenesisAlloc. The returned
// map shares storage with the recorder; callers must not mutate it concurrently
// with further Record calls.
func (r *Recorder) Alloc() types.GenesisAlloc {
	out := make(types.GenesisAlloc, len(r.accounts))
	for addr, a := range r.accounts {
		out[addr] = *a
	}
	return out
}

// Has reports whether addr has been recorded.
func (r *Recorder) Has(addr common.Address) bool {
	_, ok := r.accounts[addr]
	return ok
}

// bytesToHash left-pads short slices into a common.Hash, matching the
// normalization performed by BinaryTrie.UpdateStorage on values.
func bytesToHash(b []byte) common.Hash {
	var h common.Hash
	if len(b) >= common.HashLength {
		copy(h[:], b[:common.HashLength])
	} else {
		copy(h[common.HashLength-len(b):], b)
	}
	return h
}
