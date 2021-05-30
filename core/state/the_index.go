// Copyright 2021 orbs-network
// No license

package state

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

func (s *StateDB) TheIndex_indexContractsState(block *types.Block, contracts map[common.Address]*rlp.TheIndex_rlpContract) {
	// TODO: make sure we don't need to go over s.journal.dirties
	for addr := range s.stateObjectsDirty {
		obj, exist := s.stateObjects[addr]
		if !exist {
			continue
		}
		// TODO: we might need to look in obj.pendingStorage and obj.dirtyStorage too (although they seem to always be empty)
		if len(obj.pendingStorage) > 0 || len(obj.dirtyStorage) > 0 {
			log.Warn("THE-INDEX:assert", "pendingStorage", len(obj.pendingStorage), "dirtyStorage", len(obj.dirtyStorage))
		}
		// make sure this state object is interesting
		if !((obj.code != nil && obj.dirtyCode) || (len(obj.originStorage) > 0)) {
			continue
		}
		// add the contract
		var ok bool
		var contract *rlp.TheIndex_rlpContract
		// new contract address, add it to the map
		if contract, ok = contracts[obj.address]; !ok {
			contract = &rlp.TheIndex_rlpContract{BlockNumber: block.Header().Number}
			contracts[obj.address] = contract
		}
		// add the code to the contract
		if obj.code != nil && obj.dirtyCode {
			contract.Code = obj.code
		}
		// add the modified state keys, we might need to look in obj.pendingStorage and obj.dirtyStorage too (although they seem to always be empty)
		if len(obj.originStorage) > 0 {
			contract.States = make([]rlp.TheIndex_rlpState, 0, len(obj.originStorage))
			for key, value := range obj.originStorage {
				contract.States = append(contract.States, rlp.TheIndex_rlpState{Key: key, Value: value})
			}
		}
	}
}

func (s *StateDB) TheIndex_indexAccountChanges(block *types.Block, accounts *[]rlp.TheIndex_rplAccount) {
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
			*accounts = append(*accounts, rlp.TheIndex_rplAccount{
				Address:  obj.address,
				Balance:  obj.data.Balance,
				CodeHash: obj.data.CodeHash,
			})
		} else {
			*accounts = append(*accounts, rlp.TheIndex_rplAccount{
				Address:  obj.address,
				Balance:  obj.data.Balance,
				CodeHash: nil,
			})
		}
	}
}
