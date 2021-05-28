package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func (s *StateDB) TheIndex_indexContractsState(block *types.Block, contracts map[common.Address]*rlp.TheIndex_rlpContract) {
	// todo: make sure we don't need to go over s.journal.dirties
	for addr := range s.stateObjectsDirty {
		obj, exist := s.stateObjects[addr]
		if !exist {
			continue
		}
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
		// add the modified state keys
		if len(obj.originStorage) > 0 {
			contract.States = make([]rlp.TheIndex_rlpState, 0, len(obj.originStorage))
			for key, value := range obj.originStorage {
				contract.States = append(contract.States, rlp.TheIndex_rlpState{Key: key, Value: value})
			}
		}
	}
}
