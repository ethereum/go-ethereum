package beacon

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type mockChain struct {
	config *params.ChainConfig
	tds    map[uint64]*big.Int
}

func newMockChain() *mockChain {
	return &mockChain{
		config: new(params.ChainConfig),
		tds:    make(map[uint64]*big.Int),
	}
}

func (m *mockChain) Config() *params.ChainConfig {
	return m.config
}

func (m *mockChain) CurrentHeader() *types.Header { panic("not implemented") }

func (m *mockChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	panic("not implemented")
}

func (m *mockChain) GetHeaderByNumber(number uint64) *types.Header { panic("not implemented") }

func (m *mockChain) GetHeaderByHash(hash common.Hash) *types.Header { panic("not implemented") }

func (m *mockChain) GetTd(hash common.Hash, number uint64) *big.Int {
	num, ok := m.tds[number]
	if ok {
		return new(big.Int).Set(num)
	}
	return nil
}

func TestVerifyTerminalBlock(t *testing.T) {
	chain := newMockChain()
	chain.tds[0] = big.NewInt(10)
	chain.config.TerminalTotalDifficulty = big.NewInt(50)

	tests := []struct {
		preHeaders []*types.Header
		ttd        *big.Int
		err        error
		index      int
	}{
		// valid ttd
		{
			preHeaders: []*types.Header{
				{Number: big.NewInt(1), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(2), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(3), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(4), Difficulty: big.NewInt(10)},
			},
			ttd: big.NewInt(50),
		},
		// last block doesn't reach ttd
		{
			preHeaders: []*types.Header{
				{Number: big.NewInt(1), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(2), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(3), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(4), Difficulty: big.NewInt(9)},
			},
			ttd:   big.NewInt(50),
			err:   consensus.ErrInvalidTerminalBlock,
			index: 3,
		},
		// two blocks reach ttd
		{
			preHeaders: []*types.Header{
				{Number: big.NewInt(1), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(2), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(3), Difficulty: big.NewInt(20)},
				{Number: big.NewInt(4), Difficulty: big.NewInt(10)},
			},
			ttd:   big.NewInt(50),
			err:   consensus.ErrInvalidTerminalBlock,
			index: 3,
		},
		// three blocks reach ttd
		{
			preHeaders: []*types.Header{
				{Number: big.NewInt(1), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(2), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(3), Difficulty: big.NewInt(20)},
				{Number: big.NewInt(4), Difficulty: big.NewInt(10)},
				{Number: big.NewInt(4), Difficulty: big.NewInt(10)},
			},
			ttd:   big.NewInt(50),
			err:   consensus.ErrInvalidTerminalBlock,
			index: 3,
		},
		// parent reached ttd
		{
			preHeaders: []*types.Header{
				{Number: big.NewInt(1), Difficulty: big.NewInt(10)},
			},
			ttd:   big.NewInt(9),
			err:   consensus.ErrInvalidTerminalBlock,
			index: 0,
		},
		// unknown parent
		{
			preHeaders: []*types.Header{
				{Number: big.NewInt(4), Difficulty: big.NewInt(10)},
			},
			ttd:   big.NewInt(9),
			err:   consensus.ErrUnknownAncestor,
			index: 0,
		},
	}

	for i, test := range tests {
		fmt.Printf("Test: %v\n", i)
		chain.config.TerminalTotalDifficulty = test.ttd
		index, err := verifyTerminalPoWBlock(chain, test.preHeaders)
		if err != test.err {
			t.Fatalf("Invalid error encountered, expected %v got %v", test.err, err)
		}
		if index != test.index {
			t.Fatalf("Invalid index, expected %v got %v", test.index, index)
		}
	}
}
