// Copyright 2021 orbs-network
// No license

package state

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func (s *StateDB) TheIndex_indexContractsState(contracts map[common.Address]*rlp.TheIndex_rlpContract) {
	// TODO: make sure we don't need to go over s.journal.dirties
	for addr := range s.stateObjectsDirty {
		obj, exist := s.stateObjects[addr]
		if !exist {
			continue
		}
		// make sure this state object is interesting
		if !((obj.code != nil && obj.dirtyCode) || (len(obj.originStorage) > 0 || len(obj.dirtyStorage) > 0)) {
			continue
		}
		// new contract address, add it to the map
		if _, ok := contracts[obj.address]; !ok {
			contracts[obj.address] = &rlp.TheIndex_rlpContract{Address: obj.address}
		}
		// add the code to the contract
		if obj.code != nil && obj.dirtyCode {
			contracts[obj.address].Code = obj.code
		}
		// add the modified state keys, we might need to look in obj.pendingStorage and obj.dirtyStorage too (although they seem to always be empty)
		if len(obj.originStorage) > 0 || len(obj.dirtyStorage) > 0 {
			contracts[obj.address].States = make([]rlp.TheIndex_rlpState, 0, len(obj.originStorage)+len(obj.dirtyStorage))
			for key, value := range obj.originStorage {
				contracts[obj.address].States = append(contracts[obj.address].States, rlp.TheIndex_rlpState{Key: key, Value: value})
			}
			for key, value := range obj.dirtyStorage {
				contracts[obj.address].States = append(contracts[obj.address].States, rlp.TheIndex_rlpState{Key: key, Value: value})
			}
		}
	}
}

func (s *StateDB) TheIndex_indexAccountChanges(block *types.Block, accounts *[]rlp.TheIndex_rlpAccount) {
	// TODO: make sure we don't need to go over s.journal.dirties
	for addr := range s.stateObjectsDirty {
		obj, exist := s.stateObjects[addr]
		if !exist {
			continue
		}
		// make sure this state object is interesting
		if obj.data.Root != emptyRoot {
			continue
		}
		// add the account change
		if !bytes.Equal(obj.data.CodeHash, emptyCodeHash) {
			*accounts = append(*accounts, rlp.TheIndex_rlpAccount{
				Address:  obj.address,
				Balance:  obj.data.Balance,
				CodeHash: obj.data.CodeHash,
			})
		} else {
			*accounts = append(*accounts, rlp.TheIndex_rlpAccount{
				Address:  obj.address,
				Balance:  obj.data.Balance,
				CodeHash: nil,
			})
		}
	}
}
