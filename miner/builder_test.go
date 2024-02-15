package miner

import (
	"testing"

	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestBuilder_AddTxn_Simple(t *testing.T) {
	t.Parallel()
	config, backend := newMockBuilderConfig(t)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(true)

	res, err := builder.AddTransaction(tx1)
	require.NoError(t, err)
	require.True(t, res.Success)
	require.Len(t, builder.env.receipts, 1)

	// we cannot add the same transaction again. Note that by design the
	// function does not error but returns the SimulateTransactionResult.success = false
	res, err = builder.AddTransaction(tx1)
	require.NoError(t, err)
	require.False(t, res.Success)
	require.Len(t, builder.env.receipts, 1)
}

func TestBuilder_FillTransactions(t *testing.T) {
	t.Parallel()
	config, backend := newMockBuilderConfig(t)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(true)
	errArr := backend.TxPool().Add(types.Transactions{tx1}, false, true)
	require.NoError(t, errArr[0])

	tx2 := backend.newRandomTx(true)
	errArr = backend.TxPool().Add(types.Transactions{tx2}, false, true)
	require.NoError(t, errArr[0])

	require.NoError(t, builder.FillPending())
	require.Len(t, builder.env.receipts, 2)

	require.Equal(t, tx1.Hash(), builder.env.receipts[0].TxHash)
	require.Equal(t, tx2.Hash(), builder.env.receipts[1].TxHash)
}

func TestBuilder_BuildBlock(t *testing.T) {
	t.Parallel()

	config, backend := newMockBuilderConfig(t)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(true)

	_, err = builder.AddTransaction(tx1)
	require.NoError(t, err)

	block, err := builder.BuildBlock()
	require.NoError(t, err)
	require.NotNil(t, block)
	require.Len(t, block.Transactions(), 1)
}

func newMockBuilderConfig(t *testing.T) (*BuilderConfig, *testWorkerBackend) {
	var (
		db     = rawdb.NewMemoryDatabase()
		config = *params.AllCliqueProtocolChanges
	)
	config.Clique = &params.CliqueConfig{Period: 1, Epoch: 30000}
	engine := clique.New(config.Clique, db)

	w, backend := newTestWorker(t, &config, engine, db, 0)
	w.close()

	bConfig := &BuilderConfig{
		ChainConfig: w.chainConfig,
		Engine:      w.engine,
		EthBackend:  w.eth,
		Chain:       w.chain,
		GasCeil:     10000000,
	}
	return bConfig, backend
}
