package localpool

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type MockBC struct {
	currentBlock *types.Header
	dbs          map[common.Hash]*state.StateDB
}

func (m *MockBC) Config() *params.ChainConfig {
	return params.AllDevChainProtocolChanges
}

func (m *MockBC) CurrentBlock() *types.Header {
	return m.currentBlock
}

func (m *MockBC) GetBlock(hash common.Hash, number uint64) *types.Block {
	return nil
}

func (m *MockBC) StateAt(root common.Hash) (*state.StateDB, error) {
	state, ok := m.dbs[root]
	if !ok {
		return nil, errors.New("not found")
	}
	return state, nil
}

func (m *MockBC) SetState(root common.Hash, db *state.StateDB) {
	m.dbs[root] = db
}
