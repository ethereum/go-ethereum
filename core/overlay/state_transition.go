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

package overlay

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// TransitionState represents the progress of the Verkle transition process,
// tracking which parts of the legacy MPT-based state have already been
// migrated to the Verkle-based state.
type TransitionState struct {
	// CurrentAccountHash is the hash of the next account to be migrated
	// from the MPT state to the Verkle state. Null means nothing has been
	// migrated yet.
	CurrentAccountHash common.Hash

	// CurrentSlotHash is the hash of the next storage slot (within the
	// current account) to be migrated from the MPT state to the Verkle state.
	// This field is irrelevant if StorageProcessed is true, indicating that
	// all storage slots for the current account have already been migrated.
	CurrentSlotHash common.Hash

	// StorageProcessed indicates whether all storage slots for the current
	// account have been fully migrated. This is useful in cases where the
	// transition was interrupted before all state entries were processed
	// (e.g., due to a configured migration step limit).
	StorageProcessed bool

	// CurrentPreimageOffset is the byte offset in the preimage file from
	// which the migration should resume.
	CurrentPreimageOffset uint64

	// Started is true if the transition process has begun. Note that the
	// transition is considered started only after the MPT state referenced
	// by BaseRoot has been finalized.
	Started bool

	// Ended is true if the transition process has completed, meaning the entire
	// MPT-based state has been fully migrated. When true, the complete Ethereum
	// state is available in the Verkle state, and constructing a mixed state
	// view is no longer necessary.
	Ended bool

	// BaseRoot is the MPT root hash of the read-only base state, the original
	// state prior to Verkle activation.
	BaseRoot common.Hash
}

// InTransition returns true if the translation process is in progress.
func (ts *TransitionState) InTransition() bool {
	return ts != nil && ts.Started && !ts.Ended
}

// Transitioned returns true if the translation process has been completed.
func (ts *TransitionState) Transitioned() bool {
	return ts != nil && ts.Ended
}

// LoadTransitionState retrieves the Verkle transition state associated with
// the given state root hash from the database.
func LoadTransitionState(db ethdb.KeyValueReader, root common.Hash) (*TransitionState, error) {
	data := rawdb.ReadVerkleTransitionState(db, root)
	if len(data) == 0 {
		return nil, nil
	}
	var ts TransitionState
	if err := rlp.DecodeBytes(data, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

// StoreTransitionState serializes and writes the provided Verkle transition state
// to the database with the given state root hash.
func StoreTransitionState(db ethdb.KeyValueWriter, root common.Hash, ts *TransitionState) error {
	data, err := rlp.EncodeToBytes(ts)
	if err != nil {
		return err
	}
	rawdb.WriteVerkleTransitionState(db, root, data)
	return nil
}
