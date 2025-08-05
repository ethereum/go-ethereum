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
	"bytes"
	"encoding/gob"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// TransitionState is a structure that holds the progress markers of the
// translation process.
type TransitionState struct {
	CurrentAccountAddress *common.Address // addresss of the last translated account
	CurrentSlotHash       common.Hash     // hash of the last translated storage slot
	CurrentPreimageOffset int64           // next byte to read from the preimage file
	Started, Ended        bool

	// Mark whether the storage for an account has been processed. This is useful if the
	// maximum number of leaves of the conversion is reached before the whole storage is
	// processed.
	StorageProcessed bool

	BaseRoot common.Hash // hash of the last read-only MPT base tree
}

// InTransition returns true if the translation process is in progress.
func (ts *TransitionState) InTransition() bool {
	return ts != nil && ts.Started && !ts.Ended
}

// Transitioned returns true if the translation process has been completed.
func (ts *TransitionState) Transitioned() bool {
	return ts != nil && ts.Ended
}

// Copy returns a deep copy of the TransitionState object.
func (ts *TransitionState) Copy() *TransitionState {
	ret := &TransitionState{
		Started:               ts.Started,
		Ended:                 ts.Ended,
		CurrentSlotHash:       ts.CurrentSlotHash,
		CurrentPreimageOffset: ts.CurrentPreimageOffset,
		StorageProcessed:      ts.StorageProcessed,
	}
	if ts.CurrentAccountAddress != nil {
		addr := *ts.CurrentAccountAddress
		ret.CurrentAccountAddress = &addr
	}
	return ret
}

// LoadTransitionState retrieves the Verkle transition state associated with
// the given state root hash from the database.
func LoadTransitionState(db ethdb.KeyValueReader, root common.Hash, isVerkle bool) *TransitionState {
	var ts *TransitionState

	data, _ := rawdb.ReadVerkleTransitionState(db, root)

	// if a state could be read from the db, attempt to decode it
	if len(data) > 0 {
		var (
			newts TransitionState
			buf   = bytes.NewBuffer(data[:])
			dec   = gob.NewDecoder(buf)
		)
		// Decode transition state
		err := dec.Decode(&newts)
		if err != nil {
			log.Error("failed to decode transition state", "err", err)
			return nil
		}
		ts = &newts
	}

	// Fallback that should only happen before the transition
	if ts == nil {
		// Initialize the first transition state, with the "ended"
		// field set to true if the database was created
		// as a verkle database.
		log.Debug("no transition state found, starting fresh", "is verkle", db)

		// Start with a fresh state
		ts = &TransitionState{Ended: isVerkle}
	}
	return ts
}
