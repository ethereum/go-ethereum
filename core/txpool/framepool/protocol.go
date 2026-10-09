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

package framepool

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

const (
	recentRootTupleBytes    = 72
	maxRecentRootReferences = 16
	recentRootLength        = 8192
)

var (
	recentRootEntryDomain   = crypto.Keccak256Hash([]byte("RECENT_ROOT_ENTRY"))
	recentRootStorageDomain = crypto.Keccak256Hash([]byte("RECENT_ROOT_STORAGE"))
)

// recentRootDependency derives the committed value and ring-buffer slot from a
// shape-checked 72-byte tuple. Both hashes use fixed-width big-endian encodings.
func recentRootDependency(tuple []byte) (key, entry common.Hash) {
	var index [8]byte
	binary.BigEndian.PutUint64(index[:], binary.BigEndian.Uint64(tuple[32:40])%recentRootLength)
	return crypto.Keccak256Hash(recentRootStorageDomain[:], tuple[:32], index[:]), crypto.Keccak256Hash(recentRootEntryDomain[:], tuple)
}

func (p Prefix) validRootAge(head *types.Header) bool {
	if p.RecentRootFrame < 0 {
		return true
	}
	// A missing slot cannot validate recent roots; avoid wrapping a maximal head.
	return head.SlotNumber != nil && *head.SlotNumber != math.MaxUint64 && p.NewestRootSlot < *head.SlotNumber+1 && *head.SlotNumber+1-p.OldestRootSlot < recentRootLength
}

// checkNonces compares full storage words so malformed or exhausted sequences
// cannot pass through uint64 truncation. Protocol reads do not warm the slots.
func checkNonces(statedb *state.StateDB, tx *types.Transaction) error {
	sender := *tx.FrameSender()
	keys := tx.FrameNonceKeys()
	want := uint256.NewInt(tx.Nonce())
	for i := range keys {
		var current uint256.Int
		if keys[i].IsZero() {
			current.SetUint64(statedb.GetNonce(sender))
		} else {
			word := statedb.GetState(params.NonceManagerAddress, types.FrameTxNonceSlot(sender, &keys[i]))
			current.SetBytes(word[:])
		}
		switch current.Cmp(want) {
		case 1:
			return fmt.Errorf("%w: address %v key %s, tx: %d state: %s", core.ErrNonceTooLow, sender, keys[i].Hex(), tx.Nonce(), current.Dec())
		case -1:
			return fmt.Errorf("%w: address %v key %s, tx: %d state: %s", core.ErrNonceTooHigh, sender, keys[i].Hex(), tx.Nonce(), current.Dec())
		}
	}
	return nil
}
