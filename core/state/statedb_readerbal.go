package state

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// All states must has been cached to statedb.
func (s *StateDB) Account(addr common.Address) (*types.StateAccount, error) {
	// s.getStateObject() shouldn't be used, cause when acct is nil, it'll load account from DB and result concurrent map writes for accts
	obj := s.stateObjects[addr]
	if obj != nil {
		return &obj.data, nil
	}
	return nil, fmt.Errorf("readerWithBal account %v, not exist", addr)
}

func (s *StateDB) Code(addr common.Address, codeHash common.Hash) ([]byte, error) {
	return s.reader.Code(addr, codeHash)
}

func (s *StateDB) CodeSize(addr common.Address, codeHash common.Hash) (int, error) {
	return s.reader.CodeSize(addr, codeHash)
}

func (s *StateDB) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	stateObject := s.stateObjects[addr]
	if stateObject != nil {
		if value, cached := stateObject.originStorage[slot]; cached {
			return value, nil
		}
	}
	return common.Hash{}, nil
}
