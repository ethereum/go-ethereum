package builder

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBuilder_AddTxn_Simple(t *testing.T) {
	to := common.Address{0x01, 0x10, 0xab}

	mock := newMockBuilder(t)
	txn := mock.state.newTransfer(t, to, big.NewInt(1))

	_, err := mock.builder.AddTransaction(txn)
	require.NoError(t, err)

	mock.expect(t, expectedResult{
		txns: []*types.Transaction{
			txn,
		},
		balances: map[common.Address]*big.Int{
			to: big.NewInt(1),
		},
	})
}

func newMockBuilder(t *testing.T) *mockBuilder {
	// create a dummy header at 0
	header := &types.Header{
		Number:     big.NewInt(0),
		GasLimit:   1000000000000,
		Time:       1000,
		Difficulty: big.NewInt(1),
	}

	mState := newMockState(t)

	m := &mockBuilder{
		state: mState,
	}

	stateRef, err := mState.stateAt(mState.stateRoot)
	require.NoError(t, err)

	config := &builderConfig{
		header:   header,
		preState: stateRef,
		config:   mState.chainConfig,
		context:  m, // m implements ChainContext with panics
	}
	m.builder = newBuilder(config)

	return m
}

type mockBuilder struct {
	builder *builder
	state   *mockState
}

func (m *mockBuilder) Engine() consensus.Engine {
	panic("TODO")
}

func (m *mockBuilder) GetHeader(common.Hash, uint64) *types.Header {
	panic("TODO")
}

type expectedResult struct {
	txns     []*types.Transaction
	balances map[common.Address]*big.Int
}

func (m *mockBuilder) expect(t *testing.T, res expectedResult) {
	// validate txns
	if len(res.txns) != len(m.builder.txns) {
		t.Fatalf("expected %d txns, got %d", len(res.txns), len(m.builder.txns))
	}
	for indx, txn := range res.txns {
		if txn.Hash() != m.builder.txns[indx].Hash() {
			t.Fatalf("expected txn %d to be %s, got %s", indx, txn.Hash(), m.builder.txns[indx].Hash())
		}
	}

	// The receipts must be the same as the txns
	if len(res.txns) != len(m.builder.receipts) {
		t.Fatalf("expected %d receipts, got %d", len(res.txns), len(m.builder.receipts))
	}
	for indx, txn := range res.txns {
		if txn.Hash() != m.builder.receipts[indx].TxHash {
			t.Fatalf("expected receipt %d to be %s, got %s", indx, txn.Hash(), m.builder.receipts[indx].TxHash)
		}
	}

	// The gas left in the pool must be the header gas limit minus
	// the total gas consumed by all the transactions in the block.
	totalGasConsumed := uint64(0)
	for _, receipt := range m.builder.receipts {
		totalGasConsumed += receipt.GasUsed
	}
	if m.builder.gasPool.Gas() != m.builder.config.header.GasLimit-totalGasConsumed {
		t.Fatalf("expected gas pool to be %d, got %d", m.builder.config.header.GasLimit-totalGasConsumed, m.builder.gasPool.Gas())
	}

	// The 'gasUsed' must match the total gas consumed by all the transactions
	if *m.builder.gasUsed != totalGasConsumed {
		t.Fatalf("expected gas used to be %d, got %d", totalGasConsumed, m.builder.gasUsed)
	}

	// The state must match the expected balances
	for addr, expectedBalance := range res.balances {
		balance := m.builder.state.GetBalance(addr)
		if balance.Cmp(expectedBalance) != 0 {
			t.Fatalf("expected balance of %s to be %d, got %d", addr, expectedBalance, balance)
		}
	}
}
