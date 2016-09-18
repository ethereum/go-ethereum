package state

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// Commit commits all state changes to the database.
func Commit(state *State) (common.Hash, error) {
	root, batch := CommitBatch(state)
	return root, batch.Write()
}

// CommitBatch commits all state changes to a write batch but does not
// execute the batch. It is used to validate state changes against
// the root hash stored in a block.
func CommitBatch(state *State) (common.Hash, ethdb.Batch) {
	batch := state.Db.NewBatch()
	root, _ := stateCommit(state, batch)
	return root, batch
}

func stateCommit(state *State, db trie.DatabaseWriter) (common.Hash, error) {
	// make sure the state is flattened before committing
	state = Flatten(state)
	for address, stateObject := range state.StateObjects {
		if stateObject.remove {
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			stateObject.deleted = true
			state.Trie.Delete(stateObject.Address().Bytes()[:])
		} else if state.ownedStateObjects[address] {
			// Write any contract code associated with the state object
			if len(stateObject.code) > 0 {
				if err := db.Put(stateObject.codeHash, stateObject.code); err != nil {
					return common.Hash{}, err
				}
			}
			// Write any storage changes in the state object to its trie.
			stateObject.Update()

			// Commit the trie of the object to the batch.
			// This updates the trie root internally, so
			// getting the root hash of the storage trie
			// through UpdateStateObject is fast.
			if _, err := stateObject.trie.CommitTo(db); err != nil {
				return common.Hash{}, err
			}
			// Update the object in the account trie.
			addr := stateObject.Address()
			data, err := rlp.EncodeToBytes(stateObject)
			if err != nil {
				panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
			}
			state.Trie.Update(addr[:], data)

		}
		stateObject.dirty = false
	}
	return state.Trie.CommitTo(db)
}
