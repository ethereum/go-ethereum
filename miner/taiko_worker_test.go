package miner

import (
	"testing"

	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
)

func testGenerateWorker(t *testing.T, txCount int) *worker {
	t.Parallel()
	var (
		db     = rawdb.NewMemoryDatabase()
		config = *params.AllCliqueProtocolChanges
	)
	config.Taiko = true
	config.Clique = &params.CliqueConfig{Period: 1, Epoch: 30000}
	engine := clique.New(config.Clique, db)

	w, b := newTestWorker(t, &config, engine, db, 0)
	//defer w.close()

	for i := 0; i < txCount; i++ {
		b.txPool.Add([]*types.Transaction{b.newRandomTx(true)}, true, false)
		b.txPool.Add([]*types.Transaction{b.newRandomTx(false)}, true, false)
	}

	return w
}

func TestBuildTransactionsLists(t *testing.T) {
	w := testGenerateWorker(t, 1000)
	defer w.close()

	maxBytesPerTxList := (params.BlobTxBytesPerFieldElement - 1) * params.BlobTxFieldElementsPerBlob
	txLst, err := w.BuildTransactionsLists(
		testBankAddress,
		nil,
		240_000_000,
		uint64(maxBytesPerTxList),
		nil,
		1,
		0)
	assert.NoError(t, err)
	assert.LessOrEqual(t, 1, len(txLst))
	assert.LessOrEqual(t, txLst[0].BytesLength, uint64(maxBytesPerTxList))
}
