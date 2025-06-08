package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

func (s *StateDB) Account(addr common.Address) (*types.StateAccount, error) {
	// s.getStateObject() shouldn't be used, cause when acct is nil, it'll load account from DB and result concurrent map writes for accts
	obj := s.stateObjects[addr]
	return &obj.data, nil
}

func (s *StateDB) AccountBAL(addr common.Address) (*types.StateAccount, error) {
	panic("AccountBAL not implemented for statedb")
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
		return stateObject.GetState(slot), nil
	}
	return common.Hash{}, nil
}

func (s *StateDB) StorageBAL(addr common.Address, slot common.Hash, tr *trie.StateTrie) (common.Hash, error) {
	panic("StorageBAL not implemented for statedb")
}
